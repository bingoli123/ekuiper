#!/bin/bash

ps -ef | grep kuiperd | grep -v grep | awk '{print $2}' | xargs kill -9
