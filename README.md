# stsbug

stsbug is a simple operator that demonstrates a weakness in the current (1.18) version of 
StatefulSet where it is possible for the number of controllerrevisions to grow beyond the threshold. 
This happens when the StatefulSet is frequently updated by a controller/operator and the StatefulSet
controller is not able to create pods. 


## Usage

1. Run `make install` to install the StsBug CRD into the cluster.

2. Run `make run ENABLE_WEBHOOKS=false` to run the controller against the cluster.

3. Create a new StsBug resource by running `kubectl apply -f config/samples/demo_v1_stsbug.yaml`.

The StatefulSet will be updated roughly every 5 seconds by the StsBug controller. A non-existing
PriorityClass will prevent the StatefulSet controller from creating pods. The number of 
controllerrevisions will grow beyond the default threshold of 10.