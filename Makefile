.PHONY: build push

build:
	docker build --platform=linux/x86_64 -t aronasormannew/salin:latest .

push:
	docker push aronasormannew/salin:latest
