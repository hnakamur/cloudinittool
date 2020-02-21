New-VMSwitch -SwitchName WinNAT -SwitchType Internal
New-NetIPAddress -IPAddress 192.168.254.1 -PrefixLength 24 -InterfaceAlias "vEthernet (WinNAT)"
New-NetNat -Name WinNAT -InternalIPInterfaceAddressPrefix 192.168.254.0/24
