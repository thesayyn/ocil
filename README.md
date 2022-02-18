# ocil 
#### * abbrv. oci layout 


PoC for some layout capabilities i'd like to see in go-containerregistry


## Pull

```sh
# pull only linux arm images
go run main.go pull debian:latest --select="platform.architecture.startsWith('arm')"
# yields
# 2022/02/18 17:44:50 skipping image 0 as it does not match --select
# 2022/02/18 17:44:50 picking image [1] linux/arm
# 2022/02/18 17:44:50 picking image [2] linux/arm
# 2022/02/18 17:44:50 picking image [3] linux/arm64
# 2022/02/18 17:44:50 skipping image 4 as it does not match --select
# 2022/02/18 17:44:50 skipping image 5 as it does not match --select
# 2022/02/18 17:44:50 skipping image 6 as it does not match --select
# 2022/02/18 17:44:50 skipping image 7 as it does not match --select
```
