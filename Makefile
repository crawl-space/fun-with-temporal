all: bin
	cd worker && go build -o ../bin/worker .
	cd start && go build -o ../bin/start .
	cd finish && go build -o ../bin/finish .

bin:
	mkdir bin
