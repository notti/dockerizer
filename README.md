dockerizer
==========

Did you ever want to build a docker image just containing your favorite go
program? Well thats should be as simple as:

1. Add the binary into an empty image
2. Profit

... step 2 fails since your go program uses cgo and needs some shared libaries :(

Well ok that shouldn't be that hard, so retry:

1. Add the binary into an empty image
2. Add _all_ the needed shared libs
3. Add the correct `ld.so` loader
4. Profit

Hmm ok, but now we have to find lots of stuff and add the files to the correct
folders etc... So here is a small go-program to help with that: _dockerizer_.

Usage
-----

```Shell
go run dockerizer.go -out outfile.tar <yourbinaryhere>
```

This creates a the tarfile `outfile.tar`, which includes the binary in the root
folder, the necessary loader, and the libs. Your image can now be created with:

```Dockerfile
FROM scratch
ADD outfile.tar /
```

This also works for lot of basic commands. Multiple files can be added by running
dockerizer on all the files and adding them to your image:

```Dockerfile
FROM scratch
ADD a.tar /
ADD b.tar /
...
```

Beware that only libaries are copied over - so programs needing additional files
won't work.