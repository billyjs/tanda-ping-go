# tanda-ping-go
Tanda Internship Code Challenge - Ping Backend - Using Go

## about

This project was done in `go` as this is a language i have never used before. For the database `mongodb` was used as it is a simple NoSQL db.

Depencies for this project are;
- `mux` for easy to use router,
- `mgo` for `mongodb` driver.

## steps to build

- install go `sudo apt install golang-go`

- install mongodb `sudo apt install mongodb`

- install ruby (optional used for testing only) `sudo apt install ruby`

- set temporary $GOPATH (assuming bash) - add this to .bashrc for permanent
 ``` 
 export GOPATH=$HOME/go
 export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
 ```
 
- get dependencies
 ```
 go get github.com/gorilla/mux
 go get gopkg.in/mgo.v2
 ```

- clone or download this repo

- run server `go run server.go`

- run tests `ruby pings.rb` (optional used for testing)
