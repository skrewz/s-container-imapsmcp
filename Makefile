IMAGE_NAME=imap-mcp-server
SECRET_NAME=imap_userpass
PORT=2757

.PHONY: container-build
container-build:
	podman build -t $(IMAGE_NAME) .

.PHONY: container-run
container-run:
	@echo "Checking for Podman secret '$(SECRET_NAME)'..."
	@podman secret inspect $(SECRET_NAME) > /dev/null 2>&1 || (echo "ERROR: Podman secret '$(SECRET_NAME)' not found. Create it with: echo 'user:password' | podman secret create $(SECRET_NAME) -" && exit 1)
	podman run -d \
		--env IMAP_HOST \
		--name $(IMAGE_NAME) \
		--replace \
		--secret source=$(SECRET_NAME),target=imap_userpass \
		-p $(PORT):$(PORT) \
		$(IMAGE_NAME)

.PHONY: container-stop
container-stop:
	podman ps --filter ancestor=$(IMAGE_NAME) -q | xargs -r podman stop

.PHONY: container-clean
container-clean:
	podman rm -f $$(podman ps -a --filter ancestor=$(IMAGE_NAME) -q) 2>/dev/null || true

.PHONY: image-clean
image-clean:
	podman rmi -f $(IMAGE_NAME) 2>/dev/null || true
