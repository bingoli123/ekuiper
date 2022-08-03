#!/bin/bash

file_tar_gz=".tar.gz"
#from_image_scratch="192.168.202.61/idp/base/scratch:1.0.0"
from_image_scratch="192.168.202.61/idp/base/inspur-centos-7-golog:5.3.0"
from_image_centos="192.168.202.61/ivideo/centos7.6-log:1.0.0"
from_image_java="192.168.202.61/idp/base/openjdk-log:1.8.0_275"
from_image_nvidia="192.168.202.61/ivideo/inference-log:1.0.0-alpha.0"
from_image_web="192.168.202.61/ivideo/components/wed-sidecar:3.0.0-beta.5"
exe_components="/opt/components/"
exe_run_dir="/opt/"
exe_log_dir="/var/log/components/"
image_name=""
image_version=""
image_from=""
image_cmd=""
file_path=""
push_file_path=""
cmd_centos="CMD [\"/bin/bash\", \"./cmds/start.sh\"]"
cmd_log="CMD /sbin/crond start && ./cmds/start.sh"
cmd_java_log="CMD /usr/sbin/crond start && ./cmds/start.sh"
program_language=""

function build_golang_dockerfile() {
sudo cat>$1"/dockerfile" <<END
FROM $image_from
MAINTAINER  zcj
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo 'Asia/Shanghai' >/etc/timezone
RUN mkdir -p $exe_components
RUN mkdir -p $exe_log_dir$image_name
ADD $2 $exe_components
WORKDIR $exe_components$image_name
$image_cmd
END
}

function build_c++_dockerfile() {
sudo cat>$1"/dockerfile" <<END
FROM $image_from
MAINTAINER  zcj
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo 'Asia/Shanghai' >/etc/timezone
RUN mkdir -p $exe_components
RUN mkdir -p $exe_log_dir$image_name
ADD $2 $exe_components
WORKDIR $exe_components$image_name
$image_cmd
END
}

function build_java_dockerfile() {
sudo cat>$1"/dockerfile" <<END
FROM $image_from
MAINTAINER  zcj
ENV TZ Asia/Shanghai
RUN ln -fs /usr/share/zoneinfo/\${TZ} /etc/localtime \
    && echo \${TZ} > /etc/timezone
RUN mkdir -p $exe_components
RUN mkdir -p $exe_log_dir$image_name
ADD $2 $exe_components
WORKDIR $exe_components$image_name
$image_cmd
END
}

function build_nvidia_dockerfile() {
sudo cat>$1"/dockerfile" <<END
FROM $image_from
MAINTAINER  zcj
ENV LD_LIBRARY_PATH \$LD_LIBRARY_PATH:/opt/nvidia/deepstream/deepstream-6.0/lib/
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo 'Asia/Shanghai' >/etc/timezone
RUN mkdir -p $exe_components
RUN mkdir -p $exe_log_dir$image_name
ADD $2 $exe_components
WORKDIR $exe_components$image_name
$image_cmd
END
}

function build_web_dockerfile() {
sudo cat>$1"/dockerfile" <<END
FROM $image_from
MAINTAINER  zcj
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo 'Asia/Shanghai' >/etc/timezone
RUN mkdir -p $exe_components
RUN mkdir -p $exe_log_dir$image_name
ADD $2 $exe_components
END
}

function build_dockerfile() {
	path_pwd=$(pwd)
	file_path=$path_pwd"/build_dockerfile.sh"
	#echo $file_path $1 $2
sudo cat>$file_path<<END
#!/bin/bash
END
}

function push_dockerfile() {
	path_pwd=$(pwd)
	push_file_path=$path_pwd"/push_dockerfile.sh"
	#echo $file_path $1 $2
sudo cat>$push_file_path<<END
#!/bin/bash
END
}

function split_str_info() {
	var=$1
	var=${var//_/ }
	index=0
	for value in $var; do
		let index+=1;
		if [ $index = 1 ]; then
			image_name=$value
		elif [ $index = 2 ]; then
			image_version=$value
		elif [ $index = 5 ]; then
			str_end=${value//./ }
			str_index=0
			for str_value in $str_end; do
			let str_index+=1
			if [ $str_index = 1 ]; then
				program_language=$str_value
				if [ $str_value = "golang" ]; then
                                image_from=$from_image_scratch
                                #image_cmd="CMD [\"$exe_run_dir$image_name/service/$image_name\", \"-c\", \"$exe_run_dir$image_name/configs/config.yml\"]"
                                image_cmd=$cmd_log
                        	elif [ $str_value = "c++" ]; then
                                image_from=$from_image_centos
                                image_cmd=$cmd_log

                        	elif [ $str_value = "java" ]; then
                                image_from=$from_image_java
                                image_cmd=$cmd_java_log

                        	elif [ $str_value = "nvidia" ]; then
                                image_from=$from_image_nvidia
                                image_cmd=$cmd_log

                       		elif [ $str_value = "web" ]; then
                                image_from=$from_image_web
                                image_cmd=$cmd_centos
                        	fi
			fi
			done
		fi
	done

	#echo image_name:$image_name
	#echo image_version:$image_version
	#echo image_from:$image_from
	#echo image_cmd:$image_cmd

	#build_dockerfile $image_name $image_version
	#echo "docker build -t $image_name:$image_version ./$image_name" >> $file_path
}

function vim_dockerfile() {
	#echo $1
	split_str_info $2
echo -- $program_language -- $image_name -- $image_version -- $image_from -- $image_cmd
echo "docker build -t 192.168.202.61/ivideo/components/$image_name:$image_version ./$image_name" >> $file_path
echo "docker push 192.168.202.61/ivideo/components/$image_name:$image_version" >> $push_file_path

	if [ $program_language = "golang" ]; then
		build_golang_dockerfile $1 $2
	elif [ $program_language = "c++" ]; then
		build_c++_dockerfile $1 $2
	elif [ $program_language = "java" ]; then
		build_java_dockerfile $1 $2
	elif [ $program_language = "web" ]; then
		build_web_dockerfile $1 $2
	elif [ $program_language = "nvidia" ]; then
		build_nvidia_dockerfile $1 $2
	fi
}

function is_tar_file() {
	exec_cmd="ls $1/$2"
	#echo $exec_cmd

	for file in `$exec_cmd`
	do
		if [[ $file == *$file_tar_gz* ]]; then
			#exec_cmd=$file" include "$file_tar_gz
			vim_dockerfile $1/$2 $file
		fi

	done
}

function read_dir() {
	for file in `ls $1`
	do
		if [ -d $1"/"$file ]; then
			is_tar_file $1 $file
		fi
	done
}

function main() {
	if [ $# != 1 ]; then
		echo "please input param"
	else
		build_dockerfile
		push_dockerfile
		read_dir $1;
	fi
}

main $1
