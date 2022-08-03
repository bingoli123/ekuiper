echo "旧版本为: $1"
echo "新版本为: $2"

sed -i '' 's/'$1'/'$2'/' ./charts/helmcharts/values.yaml
sed -i '' 's/'$1'/'$2'/' ./charts/global-template.yaml
sed -i '' 's/'$1'/'$2'/' ./charts/helmcharts/Chart.yaml
sed -i '' 's/'$1'/'$2'/' ./kuiperd/packfile
sed -i '' 's/'$1'/'$2'/' pkg.sh
sed -i '' 's/'$1'/'$2'/' ../Makefile