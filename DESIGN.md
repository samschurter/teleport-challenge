# ALPS
Because every Go package requires a clever 4-letter acronym.

## Structure
Minimum Go version will be 1.16 because that's what's installed on my machine, and Go modules will be used for dependency 
management. The overall architecture of the example program implementing the ALPS library will be 2 separate 
binaries. It could be done in one binary, with the job worker also providing the CLI commands by checking to see if the 
job worker is already running first before launching a new worker or just responding to commands. However, I think that 
designing it with this structure provides for a clean, simple implementation, provides a separation of concerns, and 
makes future extensibility easier. 

- *pkg*
    - *log*: implement a common logging interface accross all components. This will not be developed from scratch,
    just adapt the logger used in other projects
    - *alps*: library code for creating and listening to Arbitrary Linux Process Service.
- *cmd*
    - *api*: HTTP API server using mTLS authentication with ACL authorization.  Authentication and authorization will be 
    done via middleware. Any endpoint will return `{"status": 401, "error": "you are not authorized to perform that 
    action"}` if the request does not have a recognized user.
        - **POST /start** starts a process and returns an ID to use for getting output and control
            - Request: 
            ```json
            {"job": "ls -lh /var/log"}
            ```
            - Success response: 
            ```json
            {
                "status": 200, 
                "jobID": 123, 
                "owner": "user@example.com",
                "startTime": "2020-08-18T12:00:00Z" 
            }
            ``` 
            - Error response: 
            ```json
            {"status": 500, "error": "some failure"}
            ```
        - **POST /stop/:jobID** stops the job with the given ID
            - Success response: 
            ```json
            {
                "status": 200, 
                "jobID": 123, 
                "owner": "user@example.com",
                "startTime": "2020-08-18T12:00:00Z", 
                "stopTime": "2020-08-18T12:00:00Z" ,
                "exitCode": 0
            }
            ```
            - Error response 
            ```json
            {"status": 500, "error": "some failure"}
            ```
        - **GET /status/:jobID** returns the status of the job with the given ID
            - Success response:
            ```json
            {
                "status": 200, 
                "jobID": 123, 
                "job": "ls -lh /var/log",
                "owner": "user@example.com",
                "startTime": "2020-08-18T12:00:00Z", 
                "jobStatus": "stopped",
                "stopTime": "2020-08-18T12:00:00Z",
                "stdoutSize": 321,
                "stderrSize": 123,
                "exitCode": 0
            }
            ```
            - Error response: 
            ```json
            {"status": 404, "error": "no job found with that ID"}
            ```
        - **GET /stderr/:jobID** returns the stderr of a job
            - Response: raw bytes
        - **GET /stdout/:jobID** returns the stdout of a job
            - Response: raw bytes 
    - *cli*: the CLI tool for managing jobs. Hard-coded paths to credential files and server address.
        - Commands will include `start`, `stop`, `status`, and `output` just like the HTTP API.
- *Makefile*: will build one or both of the executables


### Worker
The `os/exec` package allows capturing stdout/stderr directly and store it in the job. Not memory efficient to keep it 
all in RAM, but quick to implement. Storing stdout and stderr will be done separately to allow the user to get get 
them as two separate fields when retreiving output.

Worker will initialize logging, initialize worker hub. Worker hub will contain a list of all workers.

Job type will include the job owner id, the command that started it, times for different events such as start 
and finish, byte slices of stderr and stdout and other relevant information required by the API and CLI. 

Each job will be protected by a mutex to prevent conflicting commands. The jobID will be a UUIDv4 to prevent collisions 
and sequential ID attacks.

The hub will have a map of jobID to jobs. The map will also be protected by a mutex to prevent simultaneous access.

### API
Will use the `net/http` package for ease of use and fast development. Server will be configured for mTLS with TLS
version 1.3 only. The HTTP server will have a locally generated CA cert to validate the client TLS certificates, and 
middleware will perform the ACL authorization based on the certificate's Subject field before passing requests off to the 
handlers. As there are only 5 endpoints, no third-party mux will be used and path variables will be parsed manually.

#### Authentication
There will be one CA cert generated to sign all the other certs.
Then I will construct a TLSConfig with the CA certificate provided in the ClientCAs CertPool, MinVersion set to 1.3, 
and ClientAuth set to RequireAndVerifyClientCert. Then a server will be started with ListenAndServeTLS passing in the 
server certificate and secret key. That will leave the Go server to validate and accept or reject the connection.

If the connection is accepted and the server believes the certificate from the client is valid, then all requests will 
pass through an authorization middleware. That will extract the tls.ConnectionState from the http.Request and check the 
Subject field of the first PeerCertificate. The Organization from the subject will be checked against a hardcoded list 
of usernames and routes. If the username is not allowed to access that route, then the request will be rejected.

### CLI
The CLI will be built with the `github.com/spf13/cobra` library for fast development. Server address and credentials 
will be hard-coded for simplicity. The CLI will communicate with the HTTP server configured for TLS. THe client will
have the same CA cert in its cert pool for validating server certificates.

#### Examples
```
> alps start `cd /var/log && ls -lh`
Started Job 123 for user@example.com
> alps stop 123
Stopped Job 123. Output will be available for 1 hour.
```

### Testing
All testing will be done with the standard library. The ALPS library will be tested to make sure that it can perform all 
required functions. Tests will be designed to run in parallel to give the best chance for the race detector to catch any
errors. I will likely create a trivial test program to run that simply sleeps and occasionally writes predictable output
to file/STDERR/STDOUT. The API and CLI libraries will only be lightly tested as they are simply utilizing the ALPS 
library and I don't want to repeat testing efforts too much. The focus will be on testing the security (authentication 
and authorization) of the HTTP API.

## Tradeoffs
- The API will use a hard-coded "access control list" which really just consists of a few usernames mapped to access
levels. This will not be production ready and would require a lot more consideration as well as persistence.
- The `output` endpoint/command will be extremely basic and will not implmement any sort of pagination or windowing. 
The assumption will be that the ouput will be of manageable size and the list of jobs will be small.
- The CLI for this application would be a good candidate for an interactive shell tool, but implementing that would
add too much development overhead.
- There will be no persistence in the job worker. If it crashes, all running jobs along with output will be lost.
Not implementing any disk storage mechanisms will speed development and keep things simpler with consideration to 
race conditions. There will never be any question of freshness of data on disk or in memory.
- Older clients which only support TLS 1.2 maximum will not be supported in favor of the greater security of 1.3
- The CLI will use an  `http.Client` to interact with the HTTP API endpoints. More efficient communication could
be designed, but this will speed development.
- The API will not be using a 3rd party router, so features like robust path variable handling will not be possible in
the amount of time available for development.
- The API will only be implementing a simple mTLS authentication and will not, for example, be checking certificate
revocation lists.


## Plan
Develop in 3 phases. 
- Implement the worker library.
- Implement the HTTP API
- Implement the CLI