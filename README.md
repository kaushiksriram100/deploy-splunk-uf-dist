shyun master/slave setup

A wrapper to distribute ansible deployments to several VMs. I started this as a practice/learning golang. 

This wrapper uses enables start a master and several slaves. Master will open a TCP socket and send messages to slaves (the ansible properties).

Slaves will run on them separately. .
