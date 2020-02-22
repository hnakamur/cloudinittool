cloudinittool
=============

cloudinittool is a small command line tool for managing cloud-init local datasource.

It it not a general tool, but a minimal tool just for my needs to boot a Ubuntu VM on
[multipass](https://github.com/canonical/multipass) and Hyper-V.

## Usage

```
Usage: cloudinittool <subcommand> [options]

subcommands:
  modify-user-data    Modify user-data.
  make-iso            Make an ISO image

Run cloudinittool <subcommand> -h to show help for subcommand.
```

```
Usage: cloudinittool modify-user-data [options]

options:
  -in string
        input user-data yaml file. required.
  -out string
        output user-data yaml file. required.
  -passwd
        show prompt to input default user password. optional.
  -pub-key string
        add ssh public key to ssh_authorized_keys. optional.
```

```
Usage: cloudinittool make-iso [options]

options:
  -meta-data string
        input meta-data yaml file. optional.
  -network-config string
        input network-config yaml file. optional.
  -out string
        output ISO image file. required.
  -user-data string
        input user-data yaml file. required.
```

## Example

### Add `password` and `ssh_authorized_keys` to `user-data` file

The input file for `modify-user-data` subcommand is like below:

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
chpasswd:
  expire: false
```

Generate ssh key pair, for example:

```
ssh-keygen -t ed25519 -f ~/.ssh/vm.id_ed25519 -C vm -N ''
```

Run the following command to add the password and an authorized key for the default user.
You can input the password at `Password:` prompt and the `Confirm password:` prompt.

```
cloudinittool modify-user-data -in user-data.in.yml -out user-data \
  -passwd -pub-key ~/.ssh/vm.id_ed25519.pub
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
password: $6$...SALT...$...HASHED_PASSWORD_HERE...
chpasswd:
  expire: false
ssh_authorized_keys:
- |
  ssh-ed25519 ...YOUR_PUBLIC_KEY_HERE...
```

### Use user-data file when launching a VM with multipass

You can use pass this `user-data` file to [multipass launch](https://discourse.ubuntu.com/t/multipass-launch-command/10846) with the `--cloud-init` option.

```
multipass launch --name primary --cpus 2 --mem 4G --disk 100G --cloud-init user-data
```

### Make a data source ISO image for Hyper-V

An example `network-config` in [Networking Config Version 2](https://cloudinit.readthedocs.io/en/latest/topics/network-config-format-v2.html#network-config-v2) format:

```
version: 2
ethernets:
    eth0:
        dhcp4: false
        addresses:
            - 192.168.254.2/24
        gateway4: 192.168.254.1
        nameservers:
            addresses: [8.8.8.8, 8.8.4.4]
```

You can make an ISO image to pass to launch a VM on Hyper-V, for example:

```
cloudinittool make-iso -user-data user-data -network-config network-config -out cloud-init.iso
```
