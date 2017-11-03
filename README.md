#### CS 425 machine problem 2 -- Membership List  

#### Usage
To run this progam, first build all packages
```go build -o ../../pkg/member.a member/member.go;```
```go build -o ../../pkg/util.a util/util.go;```
```go build -o ../../pkg/daemon.a daemon/daemon.go```  

#### Run server 
Run ```go run membership.go```.
We have specified Port 4000 as our UDP listener.

#### Command
Enter command line input ```JOIN```,```LEAVE```,```LIST``` to join group, leave group and list all active nodes in the group.

#### Output
All outputs will be stored in vm<machineID>.log and will also shows in Standard output.