#!/bin/bash
# commit信息均为必填项，下方四个测试结果为选填项，如果不上报某些信息，请删除相应行的内容

# commit信息
commitShortSHA=${CI_COMMIT_SHA:0:8}
repositoryName=${CI_PROJECT_NAME}
branchName=$CI_COMMIT_REF_NAME
committer=$GITLAB_USER_EMAIL # 提交人，推荐使用邮箱
commitURL=https://https://gitlab.deepglint.com/deepface/${CI_PROJECT_NAME}/commit/$CI_COMMIT_SHA

# 单元测试信息
unitPass=$(grep '^--- PASS' bin/utStatistics.tmp |wc -l)
unitFail=$(grep '^--- FAIL' bin/utStatistics.tmp |wc -l)
unitTestStatistics='{"pass":'${unitPass}',"fail":'${unitFail}',"skip":0}' # 测试结果，目前仅支持包含key为pass、fail和skip的json字符串
unitTestReportFile=@bin/unitTestReport.tar.gz # 测试报告文件，目前仅支持tar.gz

# 系统测试信息
systemTestStatistics='{"pass":'$SYS_PASS',"fail":'$SYS_FAIL',"skip":0}' # 测试结果，目前仅支持包含key为pass、fail和skip的json字符串
systemTestReportFile=@ai_review.tar.gz # 测试报告文件，目前仅支持tar.gz

# 代码质量信息
codeQuality='{"high":42,"middle":13,"low":5}' # 测试结果，目前仅支持包含key为high、middle和low的json字符串
codeQualityReportFile=@pp.tar.gz # 测试报告文件，目前仅支持tar.gz

# 测试覆盖率信息
unitCoverage=$(grep  '\(statements\)' bin/ut_coverage.tmp | awk -F ' ' '{print $3}' |tr -d '%')
testCoverage='{"coverage":'${unitCoverage}'}' # 测试结果，目前仅支持包含key为coverage的json字符串
testCoverageReportFile=@bin/cover.tar.gz # 测试报告文件，目前仅支持tar.gz
echo 测试覆盖率:${testCoverage}


curl \
-F commitShortSHA=${commitShortSHA} \
-F repositoryName=${repositoryName} \
-F branchName=${branchName} \
-F committer=${committer} \
-F commitURL=${commitURL} \
-F unitTestStatistics=${unitTestStatistics} \
-F 'unitTestReport='${unitTestReportFile}';filename=unitTestReport.tar.gz' \
-F testCoverage=${testCoverage} \
-F 'testCoverageReport='${testCoverageReportFile}';filename=testCoverageReport.tar.gz' \
http://192.168.2.26:8018/upload/ci
