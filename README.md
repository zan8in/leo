# Leo
Leo is a network logon cracker which support many different services.

# Supported Protocols
- ssh

# Usage

```
leo -t [service]://[host]:[port]  # recommend
leo -h [host] (-H hostfile) -s [service] -port [port] # custom port number
leo -t [service]://[host]:[port] -l username (-L userfile) -p password (-P passfile) # recommand
```

# Example

Basic usage
```
leo -t ssh://192.168.66.120  # recommend
leo -t ssh://192.168.66.120:22122  # custom port number
leo -h 192.168.66.120 -s ssh
leo -h 192.168.66.120 -s ssh -port 22122
leo -H hosts.txt -s ssh
```

Advanced usage
```
leo -t ssh://192.168.66.120 -l root,kali -p root,123456,kali
leo -t ssh://192.168.66.120 -L users.txt -P pass.txt

leo -t ssh://192.168.66.120 -c 100
leo -t ssh://192.168.66.120 -c 100 -rl 300
leo -t ssh://192.168.66.120 -retries 5

leo -t ssh://192.168.66.120 -debug
```

## Discussion group

> For WeChat group, please add afrog personal account first, and remark "leo", and then everyone will be pulled into the afrog communication group.

<img src="https://github.com/zan8in/afrog/blob/main/images/afrog.png" width="33%" />

# Disclaimer
This tool is only for legally authorized enterprise security construction behavior. If you need to test the usability of this tool, please build a target environment by yourself.