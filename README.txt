To run:

-start server processes before client
-client runs on 1 VM/address, using separate threads to simulate multiple clients
-servers each run on a separate address (port# locally, VM remotely)

-must provide an input file declaring all addresses, with one IP:Port# per line. App will use the 1st address for the client and sequentially define servers with the remaining addresses.

Server Run Flags:
-in=input file (default nil)
-id=server process ID (default 1)
-h=if this server is the head of the chain (default false)
-t=if this server is the tail node (default false)

Example server start-up:
go run ./server.go -in=input.txt -id=1 -h=true


 Client Run Flags:
-in=input file (default nil)
-nc= number of concurrent clients to configure (default 1)
-craq=if want to permit reading from non-tail nodes (default false)

Example client start-up:
go run ./client.go -in=input.txt -nc=1 -craq=true