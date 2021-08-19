# ALPS
Because every Go package requires a clever 4-letter acronym.

## Structure
Minimum Go version will be 1.16 because that's what's installed on my machine, and Go modules will be used for dependency 
management. The overall architecture of the example program implementing the ALPS library will be 3 separate 
binaries. It could be done in one binary, with the job worker also providing the HTTPS server, and providing the
CLI commands by checking to see if the job worker is already running first before launching a new worker. However,
I think that designing it with this structure provides for a clean, simple implementation, provides a separation 
of concerns, and makes future extensibility easier. For example, it's likely that a feature such as scheduling 
jobs which occur in the future or are recurring would be developed. In that case, it would be best to have the 
job worker able to operate entirely independently of the HTTP service to isolate it from failures.

- *pkg*
    - *ipc*: types and methods required for inter-process communication.
    - *log*: implement a common logging interface accross all components. This will not be developed from scratch,
    just adapt the logger used in other projects
    - *alps*: library code for creating and listening to Arbitrary Linux Process Service.
- *cmd*
    - *api*: HTTP API server using mTLS authentication with ACL authorization. Will communicate with the job 
    worker via RPC. Authentication and authorization will be done via middleware. Any endpoint will return 
    `{"status": 401, "error": "you are not authorized to perform that action"}` if the request does not have a recognized
    user.
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
                "outputExpires": "2020-08-18T12:00:00Z", 
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
                "stopTime": "2020-08-18T12:00:00Z" ,
                "outputExpires": "2020-08-18T12:00:00Z", 
                "exitCode": 0
            }
            ```
            - Error response: 
            ```json
            {"status": 404, "error": "no job found with that ID"}
            ```
        - **GET /output/:jobID** returns the output of a job
            - Success response:
            ```json
            {
                "status": 200, 
                "jobID": 123, 
                "job": "ls -lh /var/log",
                "jobStatus": "stopped",
                "owner": "user@example.com",
                "startTime": "2020-08-18T12:00:00Z", 
                "stopTime": "2020-08-18T12:00:00Z" ,
                "outputExpires": "2020-08-18T12:00:00Z",
                "exitCode": 0
                "stderrSize": 123,
                "stderrBytes": "bytes",
                "stdoutSize": 321,
                "stdoutBytes": "more bytes"
            }
            ```
            - Error response: 
            ```json
            {"status": 404, "error": "no job found with that ID"}
            ```
        - **GET /list** gets a list of all of the user's jobs and their status. Will not return jobs 
        the user does not own.
            - Success response:
            ```json
            {
                "status": 200, 
                "jobs": [
                    {
                        "jobID": 123,
                        "job": "ls -lh /var/log",
                        "owner": "user@example.com",
                        "jobStatus": "stopped",
                        "startTime": "2020-08-18T12:00:00Z", 
                        "stopTime": "2020-08-18T12:00:00Z" ,
                        "outputExpires": "2020-08-18T12:00:00Z",
                    },
                    {
                        "jobID": 124,
                        "job": "ticker -interval=1m",
                        "owner": "user@example.com",
                        "jobStatus": "running",
                        "startTime": "2020-08-18T12:00:00Z", 
                    }
                ]
            }
            ```
            - Error response: 
            ```json
            {"status": 500, "error": "some failure"}
            ```
        - **POST /take/:jobID** will take ownership of a job away from the user that started it, as long as the 
        user issuing the command has `admin` access.
            - Success response: 
            ```json
            {
                "status": 200, 
                "jobID": 123, 
                "owner": "admin@example.com",
                "oldOwner": "user@example.com",
            }
            ``` 
            - Error response: 
            ```json
            {"status": 404, "error": "no job found with that id"}
            {"status": 403, "error": "you are forbidden to take that job: not an admin"}
            ``` 
    - *cli*: the CLI tool for managing jobs. Accepts credentials and server address and manages the connection from
    there.
        - First command run must be `config`. Will behave similarly to the AWS CLI `config` by setting some default
        connection and credential info which will be used in subsequent commmands.
        - Commands will include `start`, `stop`, `status`, `output`, `take`, and `list` just like the HTTP API.
        These commands will fail if the CLI has not been configured.
    - *worker*: The actual Linux service to manage jobs. Using a service is convenient because it provides 
    automatic restarts if the service crashes due to somebody running something that consumes enormous RAM or 
    something.
        - Exposes the methods for Start, Stop, Status, Output, List, and Take via RPC
- *Makefile*: will build one or all of the 3 executables


### Worker
The job worker will make use of `net/rpc` for communicating with the HTTP server. The package is in feature lock, but 
provides an easy interface for IPC and development is still current. There have been new commits to the package that
will be included in the Go 1.17 release. Using Unix domain sockets will allow pretty low latency and is supported by 
net/rpc. Better performance could be possible, but won't be necessary for this example. 

The `os/exec` package allows capturing stdout/stderr directly and store it in the job. Not memory efficient to keep it 
all in RAM, but quick to implement. Storing stdout and stderr will be done separately to allow the user to get get 
them as two separate fields when retreiving output.

Worker will initialize logging, initialize worker hub, and intialize an RPC listener. Worker hub will contain a list
of all workers and a go routine to occasionally prune workers that have stopped. A worker will be considered stale 
after it has been stopped for 1 hour and its output will be discarded.

Job type will include the job owner id, the command that started it, times for different events such as start 
and finish, byte slices of stderr and stdout and other relevant information required by the API and CLI. Listener
will make a few methods available via RPC for use by the API and CLI clients. The List method will only list jobs
owned by the users passed in as an argument, or all if there are no users passed in.

Each job will be protected by a mutex to prevent conflicting commands.

The hub will have a map of jobID to jobs. The map will also be protected by a mutex to prevent simultaneous access.

### API
Will use the `net/http` package for ease of use and fast development. Server will be configured for mTLS with TLS
version 1.3 only and a small set of ciphers. The HTTP server will verify the client TLS certificates and 
middleware will perform the authorization before passing requests off to the handlers. As there are only 6 
endpoints, no third-party mux will be used and path variables will be parsed manually.

### CLI
The CLI will be built with the `github.com/spf13/cobra` library for fast development. A `config` command will be
provided to allow the user to configure default credentials once instead of passing them with every command. 
This configuration will be stored in a file, but the file will be automatically generated and will require no 
management by the user. 

#### Examples
```
> alps config host 127.0.0.1:443
> alps config ca ca-cert.pem
> alps config user cert.pem key.pem
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
- The `list` and `output` endpoints/commands will be extremely basic and will not implmement any sort of 
pagination or windowing. The assumption will be that the ouput will be of manageable size and the list of jobs
will be small.
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
- Implement the worker. This will require setting up the ALPS library and all the IPC procedures
- Implement the HTTP API
- Implement the CLI