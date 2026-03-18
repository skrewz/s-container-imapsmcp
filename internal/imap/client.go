package imap

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Client struct {
	conn     *client.Client
	host     string
	port     int
	username string
	password string
}

type EmailSummary struct {
	ID       uint32
	UID      uint32
	Subject  string
	From     string
	Date     time.Time
	Preview  string
	Mailbox  string
	Size     uint32
	Seen     bool
	Answered bool
	Flagged  bool
}

type EmailListResult struct {
	Emails     []EmailSummary
	Total      int
	Limit      int
	StartIndex int
	Returned   int
	HasMore    bool
	Mailbox    string
}

type EmailContent struct {
	Summary     EmailSummary
	Headers     map[string][]string
	TextBody    string
	HTMLBody    string
	Preview     string
	Attachments []AttachmentMetadata
}

type AttachmentMetadata struct {
	Filename string
	Size     int64
	Type     string
}

func NewClient(host string, port int, username, password string) (*Client, error) {
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := client.DialTLS(fmt.Sprintf("%s:%d", host, port), tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	if err := conn.Login(username, password); err != nil {
		conn.Close()
		fmt.Printf("username: %q, len(pass): %d", username, len(password))
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	return &Client{
		conn:     conn,
		host:     host,
		port:     port,
		username: username,
		password: password,
	}, nil
}

func (c *Client) CheckConnection() error {
	_, err := c.conn.Status("", []imap.StatusItem{})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ListMailboxes(pattern string) ([]*imap.MailboxInfo, error) {
	var mailboxPattern string
	if pattern == "" {
		mailboxPattern = "%"
	} else {
		mailboxPattern = pattern
	}

	mailboxes := make(chan *imap.MailboxInfo, 100)
	go func() {
		if err := c.conn.List("", mailboxPattern, mailboxes); err != nil {
			close(mailboxes)
		}
	}()

	var result []*imap.MailboxInfo
	for mb := range mailboxes {
		result = append(result, mb)
	}

	return result, nil
}

func (c *Client) SelectMailbox(name string) error {
	_, err := c.conn.Select(name, false)
	return err
}

func (c *Client) ListEmails(mailbox string, limit, startIndex int) (*EmailListResult, error) {
	if err := c.SelectMailbox(mailbox); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	status := c.conn.Mailbox()
	if status == nil {
		return nil, fmt.Errorf("no mailbox selected")
	}

	total := int(status.Messages)
	if total == 0 {
		return &EmailListResult{
			Emails:     []EmailSummary{},
			Total:      0,
			Limit:      limit,
			StartIndex: startIndex,
			Mailbox:    mailbox,
		}, nil
	}

	if startIndex >= total {
		return &EmailListResult{
			Emails:     []EmailSummary{},
			Total:      total,
			Limit:      limit,
			StartIndex: startIndex,
			Mailbox:    mailbox,
		}, nil
	}

	end := startIndex + limit
	if end > total {
		end = total
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(uint32(startIndex+1), uint32(end))

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchUid,
		imap.FetchInternalDate,
		imap.FetchRFC822Size,
	}

	messages := make(chan *imap.Message, end-startIndex)

	go func() {
		c.conn.Fetch(seqSet, items, messages)
	}()

	var summaries []EmailSummary
	for msg := range messages {
		summary := c.parseMessage(msg)
		summary.Mailbox = mailbox
		summaries = append(summaries, summary)
	}

	returned := len(summaries)
	hasMore := (startIndex + returned) < total

	return &EmailListResult{
		Emails:     summaries,
		Total:      total,
		Limit:      limit,
		StartIndex: startIndex,
		Returned:   returned,
		HasMore:    hasMore,
		Mailbox:    mailbox,
	}, nil
}

func (c *Client) parseMessage(msg *imap.Message) EmailSummary {
	summary := EmailSummary{
		ID:      msg.SeqNum,
		UID:     msg.Uid,
		Size:    msg.Size,
		Mailbox: "INBOX",
	}

	if msg.Envelope != nil {
		summary.Subject = msg.Envelope.Subject
		summary.From = c.parseAddresses(msg.Envelope.From)
		summary.Date = msg.Envelope.Date
	}

	if msg.Flags != nil {
		for _, flag := range msg.Flags {
			switch flag {
			case imap.SeenFlag:
				summary.Seen = true
			case imap.AnsweredFlag:
				summary.Answered = true
			case imap.FlaggedFlag:
				summary.Flagged = true
			}
		}
	}

	return summary
}

func (c *Client) parseAddresses(addrs []*imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}

	var result string
	for i, addr := range addrs {
		if i > 0 {
			result += "; "
		}
		result += fmt.Sprintf("%s <%s>", addr.PersonalName, addr.MailboxName)
	}
	return result
}

func (c *Client) ReadEmail(uid uint32, mailbox string) (*EmailContent, error) {
	if err := c.SelectMailbox(mailbox); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchUid,
		imap.FetchInternalDate,
		imap.FetchRFC822Size,
		imap.FetchBodyStructure,
	}

	messages := make(chan *imap.Message, 1)

	go func() {
		c.conn.Fetch(seqSet, items, messages)
	}()

	var msg *imap.Message
	for m := range messages {
		msg = m
		break
	}

	if msg == nil {
		return nil, fmt.Errorf("email with UID %d not found in mailbox %s", uid, mailbox)
	}

	content := &EmailContent{
		Summary: c.parseMessage(msg),
	}

	if msg.BodyStructure != nil {
		textBody, htmlBody, attachments := c.extractBodyContent(msg.BodyStructure)
		content.TextBody = textBody
		content.HTMLBody = htmlBody
		content.Attachments = attachments
		content.Preview = truncateString(textBody, 200)
		if content.Preview == "" {
			content.Preview = truncateString(htmlBody, 200)
		}
	}

	return content, nil
}

func (c *Client) extractBodyContent(bs *imap.BodyStructure) (string, string, []AttachmentMetadata) {
	var textBody, htmlBody string
	var attachments []AttachmentMetadata

	if bs.MIMEType == "TEXT" {
		if strings.Contains(strings.ToLower(bs.MIMESubType), "plain") {
			textBody = "Body content available"
		} else if strings.Contains(strings.ToLower(bs.MIMESubType), "html") {
			htmlBody = "HTML content available"
		}
	} else if bs.MIMEType == "MULTIPART" {
		for _, part := range bs.Parts {
			t, h, att := c.extractBodyContent(part)
			textBody += t
			htmlBody += h
			attachments = append(attachments, att...)
		}
	} else if bs.MIMEType == "APPLICATION" || bs.MIMEType == "IMAGE" || bs.MIMEType == "AUDIO" || bs.MIMEType == "VIDEO" {
		attachments = append(attachments, AttachmentMetadata{
			Filename: bs.Params["filename"],
			Size:     int64(bs.Size),
			Type:     bs.MIMEType,
		})
	}

	return textBody, htmlBody, attachments
}

func (c *Client) SearchEmails(query, mailbox string, limit int) ([]EmailSummary, error) {
	if err := c.SelectMailbox(mailbox); err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", mailbox, err)
	}

	var searchCriteria *imap.SearchCriteria
	if query != "" {
		searchCriteria = c.parseSearchQuery(query)
	} else {
		searchCriteria = imap.NewSearchCriteria()
	}

	seqNums, err := c.conn.Search(searchCriteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	uids := seqNums
	if len(uids) == 0 {
		return []EmailSummary{}, nil
	}

	count := len(uids)
	if limit > 0 && count > limit {
		count = limit
	}

	uids = uids[:count]

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchUid,
		imap.FetchInternalDate,
		imap.FetchRFC822Size,
	}

	messages := make(chan *imap.Message, count)

	go func() {
		seqSet := new(imap.SeqSet)
		for _, uid := range uids {
			seqSet.AddNum(uid)
		}
		c.conn.Fetch(seqSet, items, messages)
	}()

	var summaries []EmailSummary
	for msg := range messages {
		summary := c.parseMessage(msg)
		summary.Mailbox = mailbox
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (c *Client) parseSearchQuery(query string) *imap.SearchCriteria {
	criteria := imap.NewSearchCriteria()

	query = strings.ToUpper(query)

	if strings.HasPrefix(query, "FROM ") {
		criteria.Header = make(map[string][]string)
		criteria.Header["From"] = []string{strings.TrimPrefix(query, "FROM ")}
	} else if strings.HasPrefix(query, "SUBJECT ") {
		criteria.Header = make(map[string][]string)
		criteria.Header["Subject"] = []string{strings.TrimPrefix(query, "SUBJECT ")}
	} else if strings.HasPrefix(query, "TO ") {
		criteria.Header = make(map[string][]string)
		criteria.Header["To"] = []string{strings.TrimPrefix(query, "TO ")}
	} else {
		criteria.Text = []string{query}
	}

	return criteria
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
