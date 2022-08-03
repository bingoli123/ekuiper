## 视频感知服务

### 生成渲染后yaml文件
helm template kuiperd --namespace=ivideo  ./kuiperd > ./kuiperd.yaml

### 不安装，仅查看渲染后内容
helm install --debug --dry-run kuiperd --namespace=ivideo --tls ./kuiperd

### 安装
helm install kuiperd --namespace=ivideo --tls ./kuiperd-3.0.0.tgz
helm install kuiperd --namespace=ivideo --tls ./kuiperd
helm upgrade --install kuiperd  --namespace=ivideo --tls ./kuiperd

### 更新
helm upgrade --tls kuiperd ./kuiperd

### 删除
helm del --purge kuiperd --tls

### 打包成tgz
helm package kuiperd

### 查看实例
helm ls --all kuiperd

### 查看资源部署实例信息
kubectl get namespace
kubectl get deploy -n ivideo
kubectl get pod -n ivideo
kubectl get service -n ivideo
kubectl get pvc -n ivideo
kubectl get cm -n ivideo
