### 七牛云SSL证书管理，支持查询到期证书，上传已到期证书并更新CDN配置，删除已到期证书

* 查询所有已到期证书
```
./qiniu -config config.yaml -env test -day 0 -mothod list
```
* 查询2天后到期证书
```
./qiniu -config config.yaml -env test -day 2 -mothod list
```
* 查询2天前到期证书
```
./qiniu -config config.yaml -env test -day -2 -mothod list
```
* 上传2天后到期证书并更新CDN配置
```
./qiniu -config config.yaml -env test -day 2 -mothod upload
```
* 删除所有已到期证书
```
./qiniu -config config.yaml -env test -mothod delete
```
