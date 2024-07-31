BASE_DIR=${shell pwd}
BIN_DIR=${BASE_DIR}/bin
LIST_DIR=${BASE_DIR}/clusterLists
LIST_FILE=byovpcClusters.txt
GET_SCRIPT=${BIN_DIR}/getClusters.sh
CLUSTER_LIST=${LIST_DIR}/${shell date +"%Y-%m-%d-%H:%M:%S"}-clusterList.txt
TEST_BIN=${BIN_DIR}/verifierTester

all: build get-list required-exports

build:
	go build -o ${TEST_BIN}

get-list:
	${GET_SCRIPT}
	awk '{print $$1}' ${LIST_FILE} >> ${CLUSTER_LIST}
	mv ${LIST_FILE} ${LIST_DIR}
	@echo "------------------------------------------------------------------------------"
	@echo "Export the cluster List"
	@echo "export CLUSTERLIST=${CLUSTER_LIST}"

required-exports:	 	
	@echo ""
	@echo "------------------------------------------------------------------------------"
	@echo "# Setup the following variables"
	@echo "# If you ran make all or make get-list CLUSTERLIST export command listed above"
	@echo "------------------------------------------------------------------------------"
	@echo export CLUSTERLIST={PATH TO LAST CLUSTERLID}
	@echo export LEGACYBIN={PATH TO LEGACY BIN 0.34.x or earlier}
	@echo export LEGACYVERSION={LEGACY BINARY VERSION}
	@echo export PROBEBINARY={PATH TO NEW CURL BASED BIN 0.35.x or newer}
	@echo export PROBEVERSION={CURL BASED BINARY VERSION}
	@echo "------------------------------------------------------------------------------"
	@echo "# To run test execute the following"
	@echo "------------------------------------------------------------------------------"
	@echo "${TEST_BIN} -legacyBinary=\$$LEGACYBIN -legacyVersion=\$$LEGACYVERSION -probeBinary=\$$PROBEBINARY -probeVersion=\$$PROBEVERSION -clusterListFile=\$$CLUSTERLIST"
	@echo ""