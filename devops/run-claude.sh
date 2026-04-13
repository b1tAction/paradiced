#!/usr/bin/env bash

SYSTEM_PROMPT="禁止使用git config修改Git配置;编写代码时使用英文注释,文档一律使用中文编写. 代码中除去本项目代码中一些特殊的中文场景(例如BUFF、事件名称等)其余都使用英文;git commit时请不要使用Co-Authored-By;"

exec claude \
    --dangerously-skip-permissions \
    --append-system-prompt "$SYSTEM_PROMPT" \
    "$@"