BAT
======

The BAT package provides a set of utility functions that can be used
to test and manipulate a ciao cluster.  These functions are just wrappers
around the ciao-cli command.  They invoke ciao-cli commands and parse
and return the output from these commands in easily consumable go
values.  Invoking the ciao-cli commands directly, rather than calling
the REST APIs exposed by ciao's various services, allows us to test
a little bit more of ciao.

Example
---------

Here's a quick example.  The following code retrieves the instances defined
on the default tenant and prints out their UUIDs and statuses.

```
	instances, err := bat.GetAllInstances(context.Background(), "")
	if err != nil {
		return err
	}
	for uuid, instance := range instances {
		fmt.Printf("%s : %s\n", uuid, instance.Status)
	}

```

The bat.GetAllInstances command calls ciao-cli instance list.



