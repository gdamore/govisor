# govisord samples

This directory is a sample configuration tree for govisord.  It only really
works on POSIX systems, as the manifests herein depend on programs like
echo and true that may not exist on Windows or other systems.

The individual manifests for each service are located in the "services"
subdirectory.  File names are not important, but each service must have a
unique name.  govisord will assume every service with a manifest in the
directory should be added.  By default it will also attempt to enable them.

To try it out, point govisor at this directory using the -dir command line
switch.  For example:

	% ../govisord -dir .

The passwd file here provides a username of "demo" with password "demo".

Note that a cert.pem and key.pem file are provided for use with TLS (mostly
for testing); the certificate is self-signed and bound to localhost,
and expires sometime in September 2025.  You should *NOT* use these files
in production but generate your own instead.

You can then use the govisor command in the ../../govisor directory to
try it out.

Enjoy.
