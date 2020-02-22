$VMName = "Ubuntu VM"

# Delete the VM if it is around
If ((Get-VM -Name $VMName).Count -gt 0) {stop-vm $VMName -TurnOff -Confirm:$false -Passthru | Remove-VM -Force}
