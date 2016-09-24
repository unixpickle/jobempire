# jobempire

*jobempire* (pronounced "Job Empire") will be tool to manage many concurrent jobs (programs, network downloads, etc.) across tens or hundreds of computers. With jobempire, you will be able to queue up a bunch of work to be done, and said work will automatically be distributed over your set of slave nodes.

I want to make jobempire work seamlessly with Go across multiple platforms. This way, a job can specify a Go program to run, and said program will be run on a slave node regardless of its OS and architecture.
