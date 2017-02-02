# Dockerfiles to create a development environment

In this section you can find some Dockerfiles to build an image that will have all necessary
dependencies to build cc-oci-runtime source code and run tests.

### Prerequisite:

Before building the docker image, please run `make-bundle-dir.sh` as the `rootfs` generated
by this script will be used while building the image.
```bash
	$ sudo ../make-bundle-dir.sh ./rootfs
```

### Build container image:

To build and run, execute the next commands:

```bash
$ sudo docker build -t clearlinux-cor -f Dockerfile.clearlinux .
$ sudo docker run -ti clearlinux-cor bash
```
