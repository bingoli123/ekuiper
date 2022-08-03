cd ..
echo "开始编译"
./release.sh

echo "编译完毕 开始打包"

cd release

pkg_name_prefix=kuiperd_3.1.0-rc.2_linux_x64_golang

chmod +x kuiperd/service/bin/kuiper
chmod +x kuiperd/service/bin/kuiperd
chmod +x kuiperd/cmds/install.sh
chmod +x kuiperd/cmds/start.sh
chmod +x kuiperd/cmds/stop.sh
chmod +x kuiperd/cmds/uninstall.sh

mv kuiperd/service/etc/kuiper.yaml kuiperd/configs/config.yml

if [ -d "/tmp/charts" ]; then
  rm -rf /tmp/charts
fi
cp -r charts /tmp/

if [ -d "/tmp/kuiperd" ]; then
  rm -rf /tmp/kuiperd
fi
cp -r kuiperd /tmp/

cd /tmp
tar zcf $pkg_name_prefix.tar.gz kuiperd
rm -rf kuiperd
mkdir kuiperd
cp -r charts kuiperd
cp $pkg_name_prefix.tar.gz kuiperd
tar zcf $pkg_name_prefix.image kuiperd
cd -
cp /tmp/$pkg_name_prefix.image .

echo "发布到vsp-release:"

scp_ip=192.168.202.12
scp_port=6233
scp_pwd=LcyiVideo3?
scp_dir=/samba/VSP-Release/VSP3.1/kuiperd/
pkg_name=$pkg_name_prefix.image
scp_pkg(){
expect <<-EOF
set timeout 10
spawn scp -P$scp_port $pkg_name root@$scp_ip:$scp_dir
expect {
    "*assword*"
        {send "$scp_pwd\r"}
    "*yes/no*"
        {
            send "yes\r"
            expect "*assword*" { send "$scp_pwd\r"}
        }
}
expect eof
EOF
}

scp_pkg