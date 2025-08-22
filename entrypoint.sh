#!/bin/sh

umask ${UMASK}

if [ "$1" = "version" ]; then
  ./openlist version
else
  # Define the target directory path for aria2 service
  # Check file of /opt/openlist/data permissions for current user
  if [ test -w ./ ]; then 
  else
    echo "Error: Current user does not have write permissions in the current directory."
    echo "Please visit https://doc.oplist.org/guide/installation/docker#for-version-after-v4-1-0 for more information."
    echo "错误：当前用户在当前目录没有写权限。"
    echo "请访问 https://doc.oplist.org/guide/installation/docker#v4-1-0-%E4%BB%A5%E5%90%8E%E7%89%88%E6%9C%AC 获取更多信息。"
    echo "Exiting..."
    exit 1
  fi

  ARIA2_DIR="/opt/service/start/aria2"
  if [ "$RUN_ARIA2" = "true" ]; then
    # If aria2 should run and target directory doesn't exist, copy it
    if [ ! -d "$ARIA2_DIR" ]; then
      mkdir -p "$ARIA2_DIR"
      cp -r /opt/service/stop/aria2/* "$ARIA2_DIR" 2>/dev/null
    fi
    runsvdir /opt/service/start &
  else
    # If aria2 should NOT run and target directory exists, remove it
    if [ -d "$ARIA2_DIR" ]; then
      rm -rf "$ARIA2_DIR"
    fi
  fi
  exec ./openlist server --no-prefix
fi
