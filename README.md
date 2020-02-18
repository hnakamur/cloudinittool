cloudinituserdatatool
=====================

cloudinituserdatatool is a small command line tool for managing cloud-init user-data.

It it not a general tool, but a minimal tool just for my needs to boot a Ubuntu VM on
[multipass](https://github.com/canonical/multipass) and Hyper-V.

## Usage

```
Usage: cloudinituserdatatool <subcommand> [options]

subcommands:
  add-ssh-key    Add ssh key to user-data yaml.
  make-iso       Make an ISO image from user-data yaml.

Run cloudinituserdatatool <subcommand> -h to show help for subcommand.
```

```
Usage: cloudinituserdatatool add-ssh-key [options]

options:
  -in string
        input user-data yaml file. required.
  -out string
        output user-data yaml file. required.
  -priv string
        user ssh private key. required.
  -pub string
        user ssh public key. required.
```

```
Usage: cloudinituserdatatool make-iso [options]

options:
  -in string
        input user-data yaml file. required.
  -out string
        output ISO image file. required.
```

## Example

The input file for `add-ssh-key` subcommand is like below:

`user-data.in.yml`

```
#cloud-config
locale: en_US.UTF8
timezone: Asia/Tokyo
package_upgrade: true
package_reboot_if_required: true
apt:
  primary:
    arches:
    - amd64
    - default
    uri: http://jp.archive.ubuntu.com/ubuntu/
password: _YOUR_PASSWORD_HERE_
chpasswd:
  expire: false
```

Note although cloud-init user-data supports a lot of [modules](https://cloudinit.readthedocs.io/en/latest/topics/modules.html), this tool supports only configurations in the above example.

The `password` in `user-data.in.yml` may be a plain text or the output of `mkpasswd`.

Run `mkpasswd` to print hash of your password, for example:

```
$ mkpasswd --method=SHA-512 --rounds=4096
Password: <input password for the default user ubuntu on VM>
```

Generate ssh key pair, for example:

```
ssh-keygen -t ed25519 -f ~/.ssh/vm.id_ed25519 -C vm -N ''
```

Run the following command to add a ssh key pair to cloud-init user-data.

```
cloudinituserdatatool add-ssh-key -in user-data.in.yml -out user-data \
  -priv ~/.ssh/vm.id_ed25519 -pub ~/.ssh/vm.id_ed25519.pub
```

The output file `user-data` is like below:

```
#cloud-config
locale: en_US.UTF8
timezone: Asia/Tokyo
package_upgrade: true
package_reboot_if_required: true
apt:
  primary:
    arches:
    - amd64
    - default
    uri: http://jp.archive.ubuntu.com/ubuntu/
password: _YOUR_PASSWORD_HERE_
chpasswd:
  expire: false
ssh_authorized_keys:
- |
  ssh-ed25519 ...
write_files:
- path: /run/priv_key
  content: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    ...
    -----END OPENSSH PRIVATE KEY-----
  permissions: "0400"
- path: /run/pub_key
  content: |
    ssh-ed25519 ...
  permissions: "0600"
```

Note the `path` of ssh key pair above is not like `/home/ubuntu/.ssh/id_ed25519*`.
This is because files in `write_files` are written before the home directory is created.
So you need to move the ssh key pair yourself after a VM starts.

You can use pass this `user-data` file to [multipass launch](https://discourse.ubuntu.com/t/multipass-launch-command/10846) with the `--cloud-init` option.

```
multipass launch --name primary --cpus 2 --mem 4G --disk 100G --cloud-init user-data
```

You can make an ISO image to pass to launch a VM on Hyper-V, for example:

```
cloudinituserdatatool make-iso -in user-data -out cloud-init.iso
```
