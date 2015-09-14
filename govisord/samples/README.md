# govisord samples

This directory is a sample configuration tree for govisord.  It only really
works on POSIX systems, as the manifests herein depend on programs like
echo and true that may not exist on Windows or other systems.

The individual manifests for each service are located in the "services"
subdirectory.  File names are not important, but each service must have a
unique name.  govisord will assume every service with a manifest in the
directory should be added.  By default it will also attempt to enable them.

To try it out, point govisor at this directory using the -d command line
switch.  For example:

	% ../govisord -d .

You can then use the govisor command in the ../../govisor directory to
try it out.

Enjoy.
