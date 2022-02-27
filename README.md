# go-edit-fstab
## What is it?
It is really cumbersome to edit the fstab via ansible with a template engine.

So this small go binary let's you allow to add, remove or edit existing paramters via cmdline flags :)

## How to use


```
# This will create a new file in tmp
# if you wish to edit it directly remove both environment values
export fstab="/etc/fstab"
export targetFstab="/etc/newFstab"

# remove mountpoint := removes mountpoint
# edit key=value := adds or edit parameter
# key can be := {device, mountPoint, fsType, options, dump, pass}
# value = any valid string, for dump and pass value will be "booleanified"
go-edit-fstab <operation1> <mountpoint> "<key>=value" <operation2> ..

# change root and boot to ro mount
go run main.go edit "/boot" options=defaults,ro edit "/" options="defaults,noatime,ro" edit "/" pass="0"
```


## State
I've just coded it in 3h and just tested it. Use with caution :)
