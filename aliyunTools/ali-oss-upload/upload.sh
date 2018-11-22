#!/bin/bash

for i in 20171106 20171116 20171120 20171125 20171208 20171214 20171222 20171228 20180108 20180120 20180124 20180130
do
    ./ali-oss-upload -source /media/keith/beats1/$i >> log/${i}.log
    sleep 10
done
