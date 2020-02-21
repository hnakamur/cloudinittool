Remove-NetNat -Name WinNAT
Remove-NetIPAddress -IPAddress 192.168.254.1 -PrefixLength 24 -InterfaceAlias "vEthernet (WinNAT)"
Remove-VMSwitch WinNAT
