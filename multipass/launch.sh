#!/bin/sh
ssh-keygen -t ed25519 -f ~/.ssh/vm.id_ed25519 -C vm
cloudinittool modify-user-data -in user-data.in.yml -out user-data -passwd -pub-key ~/.ssh/vm.id_ed25519.pub
multipass launch --name primary --cpus 2 --mem 4G --disk 100G --cloud-init user-data
multipass start primary
