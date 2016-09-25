# jobempire

jobempire (pronounced "Job Empire") will be tool to manage many concurrent jobs across tens or hundreds of computers. These jobs might be high-CPU tasks, network downloads, or anything else you please.

I want to make jobempire work seamlessly with Go across multiple platforms. Your servers should be able to run any job you queue up, regardless of OS or CPU architecture. Since Go has great built-in cross compilation, the server should be able to cross-compile Go binaries on the fly.
