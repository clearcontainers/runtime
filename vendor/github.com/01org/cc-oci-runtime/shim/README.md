# `cc-shim`

`cc-shim` is a process spawned by the runtime per container workload. The runtime 
provides the pid of the cc-shim process to containerd-shim on OCI create command.

Usage:
   cc-shim --container-id $(container_id) --proxy-sock-fd $(proxy_socket_fd) \ 
	--proxy-io-fd $(io-fd) --seq-no $(io-seq-no) --err-seq-no $(err-seq-no)

Here the $(proxy_socket_fd) is the socket fd opened by the runtime for connecting
to the proxy control socket, $(io-fd) is a per exec I/O file descriptor passed by 
the proxy to the runtime, $(io-seq-no) is the sequence number passed by the proxy
to the runtime, and (err-seq-no) is the seqence number of the error stream is the
stderr has be directed to some other location.

`cc-shim` forwards all signals to the cc-proxy process to be handled by the agent
in the VM.

The shim forwards any input received from containerd-shim to cc-proxy and 
writes any data received from the proxy on the I/O file descriptor to stdout/stderr
which is picked up by containerd-shim.

TODO:
The shim should capture the exit status of the container and exit with that exit code.
