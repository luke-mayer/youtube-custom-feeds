setup:
	docker-compose build

clean:
	docker-compose down --volumes --remove-orphans

run: clean
	docker-compose up api

test: clean
	docker-compose up test
