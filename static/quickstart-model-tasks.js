export default {
	"detection": {
		Name: "Object Detection",
		Help: "Train a model to detect bounding boxes of instances of one or more object categories in images.",
		Inputs: [{
			ID: "images",
			Name: "Images",
			DataType: "image",
			Help: "An image dataset containing example inputs.",
		}, {
			ID: "detections",
			Name: "Detection Labels",
			DataType: "detection",
			Help: "A detection dataset containing bounding box labels corresponding to each input image.",
		}],
		Models: {
			'pytorch_yolov3': {
				ID: 'pytorch_yolov3',
				Name: 'YOLOv3',
				Modes: [
					{ID: 'yolov3', Name: 'YOLOv3'},
					{ID: 'yolov3-spp', Name: 'YOLOv3-SPP'},
					{ID: 'yolov3-tiny', Name: 'YOLOv3-Tiny'},
				],
				ModeHelp: `
					YOLOv3 and YOLOv3-SPP are large models providing high accuracy (YOLOv3-SPP may provide slightly higher accuarcy).
					YOLOv3-Tiny is a small model that is fast but provides lower accuracy.
				`,
				Pretrain: [{
					ID: 'coco',
					Name: 'COCO',
				}],
			},
			'pytorch_yolov5': {
				ID: 'pytorch_yolov5',
				Name: 'YOLOv5',
				Modes: [
					{ID: 'x', Name: 'YOLOv5x'},
					{ID: 'l', Name: 'YOLOv5l'},
					{ID: 'm', Name: 'YOLOv5m'},
					{ID: 's', Name: 'YOLOv5s'},
				],
				ModeHelp: `
					Larger models like YOLOv5l and YOLOv5x provide greater accuracy but slower inference than smaller models like YOLOv5s.
				`,
				Pretrain: [{
					ID: 'coco',
					Name: 'COCO',
				}],
			},
			'pytorch_mobilenetssd': {
				ID: 'pytorch_mobilenetssd',
				Name: 'MobileNet+SSD',
				Pretrain: [{
					ID: 'voc2007',
					Name: 'VOC 2007',
				}],
			},
		},
	},
	"classification": {
		Name: "Image Classification",
		Help: "Train a model to classify images into categories.",
		Inputs: [{
			ID: "images",
			Name: "Images",
			DataType: "image",
			Help: "An image dataset containing example inputs.",
		}, {
			ID: "labels",
			Name: "Classification Labels",
			DataType: "int",
			Help: "An integer dataset containing category labels corresponding to each input image.",
		}],
		Models: {
			'pytorch_resnet34': {
				ID: 'pytorch_resnet34',
				Name: 'Resnet34',
			},
			'pytorch_efficientnet': {
				ID: 'pytorch_efficientnet',
				Name: 'EfficientNet',
			},
			'pytorch_mobilenet': {
				ID: 'pytorch_mobilenet',
				Name: 'MobileNet',
			},
			'pytorch_vgg': {
				ID: 'pytorch_vgg',
				Name: 'VGG',
			},
		},
	},
};
