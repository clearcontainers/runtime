package main

import (
	"os"
	"text/template"

	"github.com/kubernetes-incubator/cri-o/server"
	"github.com/urfave/cli"
)

var commentedConfigTemplate = template.Must(template.New("config").Parse(`
# The "crio" table contains all of the server options.
[crio]

# root is a path to the "root directory". CRIO stores all of its data,
# including container images, in this directory.
root = "{{ .Root }}"

# run is a path to the "run directory". CRIO stores all of its state
# in this directory.
runroot = "{{ .RunRoot }}"

# storage_driver select which storage driver is used to manage storage
# of images and containers.
storage_driver = "{{ .Storage }}"

# storage_option is used to pass an option to the storage driver.
storage_option = [
{{ range $opt := .StorageOptions }}{{ printf "\t%q,\n" $opt }}{{ end }}]

# The "crio.api" table contains settings for the kubelet/gRPC
# interface (which is also used by crioctl).
[crio.api]

# listen is the path to the AF_LOCAL socket on which crio will listen.
listen = "{{ .Listen }}"

# stream_address is the IP address on which the stream server will listen
stream_address = "{{ .StreamAddress }}"

# stream_port is the port on which the stream server will listen
stream_port = "{{ .StreamPort }}"

# The "crio.runtime" table contains settings pertaining to the OCI
# runtime used and options for how to set up and manage the OCI runtime.
[crio.runtime]

# runtime is a path to the OCI runtime which crio will be using.
runtime = "{{ .Runtime }}"

# runtime_host_privileged is a path to the OCI runtime which crio
# will be using for host privileged operations.
# If this string is empty, crio will not try to use the "runtime"
# for all operations.
runtime_host_privileged = "{{ .RuntimeHostPrivileged }}"

# conmon is the path to conmon binary, used for managing the runtime.
conmon = "{{ .Conmon }}"

# conmon_env is the environment variable list for conmon process,
# used for passing necessary environment variable to conmon or runtime.
conmon_env = [
{{ range $env := .ConmonEnv }}{{ printf "\t%q,\n" $env }}{{ end }}]

# selinux indicates whether or not SELinux will be used for pod
# separation on the host. If you enable this flag, SELinux must be running
# on the host.
selinux = {{ .SELinux }}

# seccomp_profile is the seccomp json profile path which is used as the
# default for the runtime.
seccomp_profile = "{{ .SeccompProfile }}"

# apparmor_profile is the apparmor profile name which is used as the
# default for the runtime.
apparmor_profile = "{{ .ApparmorProfile }}"

# cgroup_manager is the cgroup management implementation to be used
# for the runtime.
cgroup_manager = "{{ .CgroupManager }}"

# The "crio.image" table contains settings pertaining to the
# management of OCI images.
[crio.image]

# default_transport is the prefix we try prepending to an image name if the
# image name as we receive it can't be parsed as a valid source reference
default_transport = "{{ .DefaultTransport }}"

# pause_image is the image which we use to instantiate infra containers.
pause_image = "{{ .PauseImage }}"

# pause_command is the command to run in a pause_image to have a container just
# sit there.  If the image contains the necessary information, this value need
# not be specified.
pause_command = "{{ .PauseCommand }}"

# signature_policy is the name of the file which decides what sort of policy we
# use when deciding whether or not to trust an image that we've pulled.
# Outside of testing situations, it is strongly advised that this be left
# unspecified so that the default system-wide policy will be used.
signature_policy = "{{ .SignaturePolicyPath }}"

# insecure_registries is used to skip TLS verification when pulling images.
insecure_registries = [
{{ range $opt := .InsecureRegistries }}{{ printf "\t%q,\n" $opt }}{{ end }}]

# The "crio.network" table contains settings pertaining to the
# management of CNI plugins.
[crio.network]

# network_dir is is where CNI network configuration
# files are stored.
network_dir = "{{ .NetworkDir }}"

# plugin_dir is is where CNI plugin binaries are stored.
plugin_dir = "{{ .PluginDir }}"
`))

// TODO: Currently ImageDir isn't really used, so we haven't added it to this
//       template. Add it once the storage code has been merged.

var configCommand = cli.Command{
	Name:  "config",
	Usage: "generate crio configuration files",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "default",
			Usage: "output the default configuration",
		},
	},
	Action: func(c *cli.Context) error {
		// At this point, app.Before has already parsed the user's chosen
		// config file. So no need to handle that here.
		config := c.App.Metadata["config"].(*server.Config)
		if c.Bool("default") {
			config = server.DefaultConfig()
		}

		// Output the commented config.
		return commentedConfigTemplate.ExecuteTemplate(os.Stdout, "config", config)
	},
}
