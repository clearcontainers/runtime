# cc-oci-runtime metrics tests

### Prerequisites
Some tests require you to enable the debug mode of cc-oci-runtime. To enable it, please take a look at:
https://github.com/01org/cc-oci-runtime#18running-under-docker

To run the metrics, just run:

```bash
$ cd tests/metrics
$ ./run_docker_metrics
```

This will run all metrics tests and generete a `results` directory.

Each file in the `result` directory contains the result for a test and has the format of a CSV.
At the end of each file you will find the result average of all the data collected by the test.

You can also run each tests script separately. e.g.

```bash
$ cd tests/metrics
$ bash workload_time/cor_create_time.sh <times_to_run>
```
