run: tidy swagger
	go run .

swagger:
	swag init

tidy: 
	go mod tidy

build:
	go build -o phantomias .

update-submodule:
	git submodule update --remote