# phunter (PHP Process Hunter)

>I want to run phpspy to collect stack trace information of a
>running php process in a distributed linux environment when it exceeds a certain memory
>or CPU threshold for a specified duration.

This tool provides a way to run [phpspy](https://github.com/adsr/phpspy) against php processes that exceed the
specified CPU and/or memory thresholds.

This tool works but has not been battle-hardened and there's a few pieces of work to complete before 
I would encourage general usage. See the [TODO list](#todo-list).

You can run it but in its current state I would recommend it's only used as a reference.

# Motivation
It's difficult to capture information on php processes running in Kubernetes if they fail to finish.
Most APM tools seem to be designed around the process finishing. This is so they can post a payload at the end of the request.

# How it works
phunter uses phpspy to provide additional functionality around watching processes running in a linux environment.

It runs as a daemon and will check all the running processes that are obtained from the **process_command** at the
defined **check_interval**. When a process has exceeded the threshold and met the threshold parameters phpspy
will be run against the process and the trace written to a file.

The traces are also available over a file server on port 9000.

# Deployment on Kubernetes

An example of deploying phunter on Kubernetes has been included. See [k8s-daemonset-example.yml](k8s-daemonset-example.yml)

# Configuration

See the [example configuration](config-example.yml)

# TODO List <a name="todo-list"></a>
1. Support all phpspy configuration options
2. Include phpspy in the phunter binary
3. Use the Docker SDK for Go
4. Set default configuration to avoid user error
5. Setup integration tests
6. Persist traces outside of the container
7. Support for applications other than php?
8. Add some more unit tests
