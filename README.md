SkyhookML
=========

SkyhookML is a platform for computer vision, providing an easy-to-use interface that makes machine learning methods for image and video data more accessible.

Website: https://www.skyhookml.org


Features
--------

- Import unlabeled image/video data, as well as annotations in YOLO, COCO, and per-class-folders formats.
- Annotate image or video for object detection, image classification, image segmentation, and object tracking tasks.
- Train various models on labeled datasets, including YOLOv5 for object detection and ResNet-34 for image classification.
- Data augmentation: random cropping, random resize, random flipping, etc.
- Apply pre-trained or custom trained models on new datasets.
- Build flexible ML execution pipelines that combine training, inference, and post-processing steps.
- Add custom Python code into pipelines for pre/post-processing, or use built-in image/video rendering, filter, union, and other operations.
- Easily combine model components to build new model architectures for joint training tasks, e.g., object detection plus image classification.


Quickstart
----------

The fastest way to get started is with the all-in-one Docker container.
First, install [nvidia-docker](https://github.com/NVIDIA/nvidia-docker); on Ubuntu:

	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
	sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
	distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
	curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
	curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list
	sudo apt update && sudo apt install -y docker-ce docker-ce-cli containerd.io nvidia-container-toolkit
	sudo systemctl restart docker

Then:

	git clone https://github.com/skyhookml/skyhookml.git
	cd skyhookml
	mkdir -p data/items data/models
	docker/allinone/run.sh

Access your deployment at http://localhost:8080.

Note: If you get an error like `nvidia-container-cli: initialization error`, make sure NVIDIA driver is installed (e.g., `sudo apt install nvidia-driver-460`; driver version must be >= 450).


Overview
--------

SkyhookML provides a web interface for easily developing ML pipelines.

### Datasets

All data in Skyhook is stored in some dataset. A dataset is a key-value map, where keys are strings and values can take on a variety of different types. For example, object detection involves inputting a dataset of image values (where keys may be filenames) and producing a dataset of object detection values -- each value in the output dataset is a list of objects detected in an image, with the key corresponding to the image filename.

Skyhook provides conversion operations to import data from a variety of formats, including YOLO text files and COCO JSON format for object detection, and per-category-folders for image classification. It can export to these formats as well.

### Annotation

Labels are annotated by creating a new dataset of labels (e.g., bounding boxes or image classes) under keys that correspond to items in an input dataset, which typically contains images or video.

Skyhook provides annotation tools for object detection, image classification, image segmentation, and object tracking tasks. Annotations can be imported from external tools such as [cvat](https://github.com/openvinotoolkit/cvat) as well.

### Operations and Pipelines

Operations transform one or more input datasets into one or more output datasets. For example, an object detection training operation takes a dataset of images and a dataset of object detection labels, and produces a one-item dataset containing trained model parameters.

In Skyhook, operations are connected to form an execution pipeline graph. This is just a graph of potentially many operations combined together to implement a more complex task. Skyhook handles efficiently re-executing the pipeline when certain operations have been modified.

### Training

Skyhook includes training operations for:

- Object detection: [YOLOv3](https://github.com/ultralytics/yolov3), [YOLOv5](https://github.com/ultralytics/yolov5), [MobileNet+SSD](https://github.com/qfgaohao/pytorch-ssd)
- Image classification: ResNet-34

Pre-trained models for most operations are available, which can be used either as initial parameters for fine-tuning or as a model for inference when dataset-specific labels are not available.

### Custom Model Architectures

Built-in model components can be easily combined together and configured to form new model architectures. For example, combine simple-backbone (an image encoder network that applies strided convolutional layers), yolov3-head (the last few layers of YOLOv3), and cls-head (softmax output with cross entropy loss) to form a network that can be jointly trained for object detection and image classification.


About
-----

- Website: https://www.skyhookml.org
