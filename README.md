# wfl - A Simple and Pluggable Workflow Language for Go

_Don't mix wfl with [WFL](https://en.wikipedia.org/wiki/Work_Flow_Language)._

[![CircleCI](https://circleci.com/gh/dgruber/wfl/tree/master.svg?style=svg)](https://circleci.com/gh/dgruber/wfl/tree/master)
[![codecov](https://codecov.io/gh/dgruber/wfl/branch/master/graph/badge.svg)](https://codecov.io/gh/dgruber/wfl)

Creating process, container, pod, task, or job workflows based on raw interfaces of
operating systems, Docker, Kubernetes, Cloud Foundry, and HPC job schedulers can be
a tedios. Lots of repeating code is required. All workload management systems have a
different API.

_wfl_ abstracts away from the underlying details of the processes, containers, and 
workload management systems. _wfl_ provides a simple, unified interface which allows
to quickly define and execute a job workflow and change between different execution
backends without changing the workflow itself.

_wfl_ does not come with many features but is simple to use and enough to define and
run jobs and job workflows with inter-job dependencies.

In its simplest form a process can be started and waited for:

```go
    wfl.NewWorkflow(wfl.NewProcessContext()).Run("convert", "image.jpg", "image.png").Wait()
```

Running a job as a Docker container requires a different context (and the image
already pulled before).

```go
    ctx := wfl.NewDockerContextByCfg(wfl.DockerConfig{DefaultDockerImage: "golang:latest"})
    wfl.NewWorkflow(ctx).Run("sleep", "60").Wait()
```

Starting a Docker container without a _run command_ which exposes ports requires more
configuration which can be provided by using a _JobTemplate_ together with the _RunT()_
method.

```go
    jt := drmaa2interface.JobTemplate{
        JobCategory: "swaggerapi/swagger-editor",
    }
    jt.ExtensionList = map[string]string{"exposedPorts": "80:8080/tcp"}
    
    wfl.NewJob(wfl.NewWorkflow(wfl.NewDockerContext())).RunT(jt).Wait()
```

Starting a Kubernetes batch job and waiting for its end is not much different.

```go
    wfl.NewWorkflow(wfl.NewKubernetesContext()).Run("sleep", "60").Wait()
```

_wfl_ aims to work for any kind of workload. It works on a Mac and Raspberry Pi the same way
as on a high-performance compute cluster. Things missing: On small scale you probably miss data
management - moving results from one job to another. That's deliberately not implemented.
On large scale you are missing checkpoint and restart functionality or HA of the workflow 
process itself.

_wfl_ works with simple primitives: *context*, *workflow*, *job*, and *jobtemplate*

Jobs can also be processed in [streams](https://github.com/dgruber/wfl/blob/master/examples/stream/stream.go).

### Getting Started

Dependencies of _wfl_ (like drmaa2) are vendored in. The only external package required to be installed
manually is the _drmaa2interface_.

```go
    go get github.com/dgruber/drmaa2interface
```

## Context

A context defines the execution backend for the workflow. Contexts can be easily created
with the _New_ functions which are defined in the _context.go_ file.

For creating a context which executes the jobs of a workflow in operating system processses use:

```go
    wfl.NewProcessContext()
```

If the workflow needs to be executed in containers the _DockerContext_ can be used: 

```go
    wfl.NewDockerContext()
```

If the Docker context needs to be configured with a default Docker image 
(when Run() is used or RunT() without a configured _JobCategory_ (which _is_ the Docker image))
then the _ContextByCfg()_ can be called.

```go
    wfl.NewDockerContextByCfg(wfl.DockerConfig{DefaultDockerImage: "golang:latest"})
```

When you want to run the workflow as Cloud Foundry tasks the _CloudFoundryContext_ can be used:

```go
    wfl.NewCloudFoundryContext()
```

Without a config it uses following environment variables to access the Cloud Foundry cloud controller API:

* CF_API (like https://api.run.pivotal.io)
* CF_USER
* CF_PASSWORD

Contexts for other workload managers like Kubernetes, DRMAA compatible HPC schedulers,
etc. will be supported when the DRMAA2 job tracker implementation is available.

For submitting Kubernetes batch jobs a Kubernetes context exists.

```go
   ctx := wfl.NewKubernetesContext()
```

Note that each job requires a container image specified which can be done by using
the JobTemplate's JobCategory. When the same container image is used within the whole
job workflow it makes sense to use the Kubernetes config.

```go
   ctx := wfl.NewKubernetesContextByCfg(wfl.KubernetesConfig{DefaultImage: "busybox:latest"})
```


## Workflow

A workflow encapsulates a set of jobs using the same backend (context). Depending on the execution
backend it can be seen as a namespace. 

It can be created by using:

```go
    wf := wfl.NewWorkflow(ctx)
```

Errors during creation can be catched with

```go
    wf := wfl.NewWorkflow(ctx).OnError(func(e error) {panic(e)})
```

or with

```go
    if wf.HasError() {
        panic(wf.Error())
    }
```

## Job

Jobs are the main objects in _wfl_. A job defines helper methods. Many of them return the job object itself to allow chaining calls in an easy way. A job can also be seen as a container and control unit for tasks.

Methods can be classified in blocking, non-blocking, job template based, function based, and error handlers.

Job submission:

* Run() -> Starts a process, container, or submits a task and comes back immediately
* RunT() -> Like above but with a JobTemplate as parameter
* Resubmit() -> Run().Run().Run()...
* RunEvery() -> Submits a task every d time.Duration
* RunEveryT()

Job control:

* Suspend() -> Stops a task from execution (e.g. sending SIGTSTP to the process)...
* Resume() -> Continues a task (e.g. sending SIGCONT)...
* Kill()

Function execution:

* Do() -> Executes a function
* Then() -> Waits for end of process and executes function
* OnSuccess() -> Executes a function if the task run successfully (exit code 0)
* OnFailure() -> Executes a function if the task failed (exit code != 0)
* OnError() -> Executes a function if the task could not be created

Blocker:

* After()
* Wait()
* Synchronize()

Job flow control:

* ThenRun() // wait() + run()
* ThenRunT()
* OnSuccessRun() // wait() + success() + run()
* OnFailureRun()
* Retry() // wait() + !success() + resubmit() + wait() + !success() ...
* AnyFailed() // checks if one of the tasks in the job failed

Job status and general checks:

* JobID() -> Returns the ID of the submitted job.
* JobInfo() -> Returns the DRMAA2 JobInfo of the job. 
* Template() 
* State()
* LastError()
* Failed()
* Success()
* ExitStatus()

## JobTemplate

JobTemplates are specifying the details about a job. In the simplest case the job is specified by the application name and its arguments like it is typically done in the OS shell. In that case the _Run()_ methods (_ThenRun()_, _OnSuccessRun()_, _OnFailureRun()_) can be used. If more details for specifying the jobs are required the _RunT()_ methods needs to be used.
I'm using currently the [DRMAA2 Go JobTemplate](https://github.com/dgruber/drmaa2interface/blob/master/jobtemplate.go)
as parameters for them. In most cases only _RemoteCommand_, _Args_, _WorkingDirectory_, _JobCategory_, _JobEnvironment_,  _StageInFiles_ are evaluated. Functionality and semantic is up to the underlying [drmaa2os job tracker](https://github.com/dgruber/drmaa2os/tree/master/pkg/jobtracker).

- [For the process mapping see here](https://github.com/dgruber/drmaa2os/tree/master/pkg/jobtracker/simpletracker)
- [For the Docker mapping here](https://github.com/dgruber/drmaa2os/tree/master/pkg/jobtracker/dockertracker)
- [For the Cloud Foundry Task mapping here](https://github.com/dgruber/drmaa2os/blob/master/pkg/jobtracker/cftracker)
- [For the Kubernetes batch job mapping here](https://github.com/dgruber/drmaa2os/blob/master/pkg/jobtracker/kubernetestracker)

The [_Template_](https://github.com/dgruber/wfl/blob/master/template.go) object provides helper functions for job templates and required as generators of job [streams](https://github.com/dgruber/wfl/blob/master/examples/stream/stream.go). For an example see [here](https://github.com/dgruber/wfl/tree/master/examples/template/template.go).

# Examples

For examples please have a look into the examples directory. [template](https://github.com/dgruber/wfl/tree/master/examples/template/template.go) is a canonical example of a pre-processing job, followed by parallel execution, followed by a post-processing job.

[test](https://github.com/dgruber/wfl/blob/master/test/test.go) is an use case for testing. It compiles
all examples with the local go compiler and then within a Docker container using the _golang:latest_ image
and reports errors.

[cloudfoundry](https://github.com/dgruber/wfl/blob/master/examples/cloudfoundry/cloudfoundry.go) demonstrates how a Cloud Foundry taks can be created.


## Creating a Workflow which is Executed as OS Processes

The allocated context defines which workload management system / job execution backend is used.

```go
    ctx := wfl.NewProcessContext()
```

Different contexts can be used within a single program. That way multi-clustering potentially
over different cloud solutions is supported.

Using a context a workflow can be established.

```go
    wfl.NewWorkflow(wfl.NewProcessContext())
```

Handling an error during workflow generation can be done by specifying a function which 
is only called in the case of an error.

```go
    wfl.NewWorkflow(wfl.NewProcessContext()).OnError(func(e error) {
		panic(e)
	})
```

The workflow is used in order to instantiate the first job using the _Run()_ method.

```go
    wfl.NewWorkflow(wfl.NewProcessContext()).Run("sleep", "123")
```

But you can also create an initial job like that:

```go
    job := wfl.NewJob(wfl.NewWorkflow(wfl.NewProcessContext()))
```

For more detailed settings (like resource limits) the DRMAA2 job template can be used as parameter for _RunT()_.

Jobs allow the execution of workload as well as expressing dependencies.

```go
    wfl.NewWorkflow(wfl.NewProcessContext()).Run("sleep", "2").ThenRun("sleep", "1").Wait()
```

The line above executes two OS processes sequentially and waits until the last job in chain is finished.

In the following example the two sleep processes are executed in parallel. _Wait()_ only waitf for the sleep 1 job. Hence sleep 2 still runs after the wait call comes back.

```go
    wfl.NewWorkflow(wfl.NewProcessContext()).Run("sleep", "2").Run("sleep", "1").Wait()
```

Running two jobs in parallel and waiting until all jobs finished can be done _Synchronize()_.

```go
    wfl.NewWorkflow(wfl.NewProcessContext()).Run("sleep", "2").Run("sleep", "1").Synchronize()
```

Jobs can also be suspended (stopped) and resumed (continued) - if supported by the execution backend (like OS, Docker).

```go
    wf.Run("sleep", "1").After(time.Millisecond * 100).Suspend().After(time.Millisecond * 100).Resume().Wait()
```

The exit status is available as well. _ExitStatus()_ blocks until the previously submitted job is finished.

```go
    wfl.NewWorkflow(ctx).Run("echo", "hello").ExitStatus()
```

In order to run jobs depending on the exit status the _OnFailure_ and _OnSuccess_ methods can be used:

```go
    wf.Run("false").OnFailureRun("true").OnSuccessRun("false")
```

For executing a function on a submission error _OnError()_ can be used.

More methods can be found in the sources.

For missing functionality or bugs please open an issue on github. Contributions welcome.
