import json
import numpy
import os, os.path
import sys

base_path = os.getcwd()
os.chdir('./lib/darknet')
sys.path.append('./')
import darknet

def eprint(s):
	sys.stderr.write(str(s) + "\n")
	sys.stderr.flush()

in_dataset_id = int(sys.argv[1])
batch_size = int(sys.argv[2])
width = int(sys.argv[3])
height = int(sys.argv[4])
threshold = 0.05

train_dir = os.path.join(base_path, 'items', str(in_dataset_id))
config_path = os.path.join(train_dir, 'yolov3.cfg')
meta_path = os.path.join(train_dir, 'obj.data')
weight_path = os.path.join(train_dir, 'yolov3.weights')

# create a new config where the width/height are set correctly
with open(config_path, 'r') as f:
	tmp_config_buf = ''
	for line in f.readlines():
		line = line.strip()
		if line.startswith('width='):
			line = 'width={}'.format(width)
		if line.startswith('height='):
			line = 'height={}'.format(height)
		tmp_config_buf += line + "\n"
tmp_config_path = '/tmp/yolov3-{}.cfg'.format(os.getpid())
with open(tmp_config_path, 'w') as f:
	f.write(tmp_config_buf)

net, class_names, _ = darknet.load_network(tmp_config_path, meta_path, weight_path, batch_size=batch_size)

os.remove(tmp_config_path)

stdin = sys.stdin.detach()
while True:
	buf = stdin.read(batch_size*width*height*3)
	if not buf:
		break

	arr = numpy.frombuffer(buf, dtype='uint8').reshape((batch_size, height, width, 3))
	arr = arr.transpose((0, 3, 1, 2))
	arr = numpy.ascontiguousarray(arr.flat, dtype='float32')/255.0
	darknet_images = arr.ctypes.data_as(darknet.POINTER(darknet.c_float))
	darknet_images = darknet.IMAGE(width, height, 3, darknet_images)
	raw_detections = darknet.network_predict_batch(net, darknet_images, batch_size, width, height, threshold, 0.5, None, 0, 0)
	detections = []
	for idx in range(batch_size):
		num = raw_detections[idx].num
		raw_dlist = raw_detections[idx].dets
		darknet.do_nms_obj(raw_dlist, num, len(class_names), 0.45)
		raw_dlist = darknet.remove_negatives(raw_dlist, class_names, num)
		dlist = []
		for cls, score, (cx, cy, w, h) in raw_dlist:
			dlist.append({
				'Category': cls,
				'Score': float(score),
				'Left': int(cx-w/2),
				'Right': int(cx+w/2),
				'Top': int(cy-h/2),
				'Bottom': int(cy+h/2),
			})
		detections.append(dlist)
	darknet.free_batch_detections(raw_detections, batch_size)
	print('json'+json.dumps(detections), flush=True)
