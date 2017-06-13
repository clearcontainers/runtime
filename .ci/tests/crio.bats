#!/usr/bin/env bats

load helpers

function teardown() {
	cleanup_test
}

@test "ctr not found correct error message" {
	start_crio
	run crioctl ctr status --id randomid
	echo "$output"
	[ "$status" -eq 1 ]
	[[ "$output" =~ "container with ID starting with randomid not found" ]]

	stop_crio
}

@test "ctr termination reason Completed" {
	start_crio
	run crioctl pod run --config "$TESTDATA"/sandbox_config.json
	echo "$output"
	[ "$status" -eq 0 ]
	pod_id="$output"
	run crioctl ctr create --config "$TESTDATA"/container_config.json --pod "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	ctr_id="$output"
	run crioctl ctr start --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr status --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" =~ "Reason: Completed" ]]

	cleanup_ctrs
	cleanup_pods
	stop_crio
}

@test "ctr termination reason Error" {
	start_crio
	run crioctl pod run --config "$TESTDATA"/sandbox_config.json
	echo "$output"
	[ "$status" -eq 0 ]
	pod_id="$output"
	errorconfig=$(cat "$TESTDATA"/container_config.json | python -c 'import json,sys;obj=json.load(sys.stdin);obj["command"] = ["false"]; json.dump(obj, sys.stdout)')
	echo "$errorconfig" > "$TESTDIR"/container_config_error.json
	run crioctl ctr create --config "$TESTDIR"/container_config_error.json --pod "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	ctr_id="$output"
	run crioctl ctr start --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr status --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	[[ "$output" =~ "Reason: Error" ]]

	cleanup_ctrs
	cleanup_pods
	stop_crio
}

@test "ctr remove" {
	start_crio
	run crioctl pod run --config "$TESTDATA"/sandbox_config.json
	echo "$output"
	[ "$status" -eq 0 ]
	pod_id="$output"
	run crioctl ctr create --config "$TESTDATA"/container_redis.json --pod "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	ctr_id="$output"
	run crioctl ctr start --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr remove --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl pod stop --id "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl pod remove --id "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	cleanup_ctrs
	cleanup_pods
	stop_crio
}

@test "ctr lifecycle" {
	start_crio
	run crioctl pod run --config "$TESTDATA"/sandbox_config.json
	echo "$output"
	[ "$status" -eq 0 ]
	pod_id="$output"
	run crioctl pod list
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr create --config "$TESTDATA"/container_redis.json --pod "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	ctr_id="$output"
	run crioctl ctr list
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr status --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr start --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr status --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr list
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr stop --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr status --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr list
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr remove --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr list
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl pod stop --id "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl pod list
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr list
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl pod remove --id "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl pod list
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr list
	echo "$output"
	[ "$status" -eq 0 ]
	cleanup_ctrs
	cleanup_pods
	stop_crio
}

@test "ctr logging" {
	start_crio
	run crioctl pod run --config "$TESTDATA"/sandbox_config.json
	echo "$output"
	[ "$status" -eq 0 ]
	pod_id="$output"
	run crioctl pod list
	echo "$output"
	[ "$status" -eq 0 ]

	# Create a new container.
	newconfig=$(mktemp --tmpdir crio-config.XXXXXX.json)
	cp "$TESTDATA"/container_config_logging.json "$newconfig"
	sed -i 's|"%shellcommand%"|"echo here is some output \&\& echo and some from stderr >\&2"|' "$newconfig"
	run crioctl ctr create --config "$newconfig" --pod "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	ctr_id="$output"
	run crioctl ctr start --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl ctr stop --id "$ctr_id"
	echo "$output"
	# Ignore errors on stop.
	run crioctl ctr status --id "$ctr_id"
	[ "$status" -eq 0 ]
	run crioctl ctr remove --id "$ctr_id"
	echo "$output"
	[ "$status" -eq 0 ]

	# Check that the output is what we expect.
	logpath="$DEFAULT_LOG_PATH/$pod_id/$ctr_id.log"
	[ -f "$logpath" ]
	echo "$logpath :: $(cat "$logpath")"
	grep -E "^[^\n]+ stdout here is some output$" "$logpath"
	grep -E "^[^\n]+ stderr and some from stderr$" "$logpath"

	run crioctl pod stop --id "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]
	run crioctl pod remove --id "$pod_id"
	echo "$output"
	[ "$status" -eq 0 ]

	cleanup_ctrs
	cleanup_pods
	stop_crio
}
