import sys
sys.path.append('./python')
import skyhook.common as lib

import json
import numpy
import os, os.path
import requests
import skimage.io, skimage.transform

import torch
import torch.optim
import torch.utils

import skyhook.pytorch.model as model
import skyhook.pytorch.util as util

import skyhook.pytorch.dataset as skyhook_dataset
import skyhook.pytorch.augment as skyhook_augment

url = sys.argv[1]
local_port = int(sys.argv[2])
batch_size = int(sys.argv[3])

local_url = 'http://127.0.0.1:{}'.format(local_port)

# Get parameters.
resp = requests.get(local_url + '/config')
config = resp.json()

params = config['Params']
arch = config['Arch']
comps = config['Components']
datasets = config['Inputs']
parent_models = config['ParentModels']
out_dataset_id = config['Output']['ID']
train_split = config['TrainSplit']
valid_split = config['ValidSplit']

arch = arch['Params']

# overwrite parameters in arch['Components'][idx]['Params'] with parameters
# from params['Components'][idx]
if params.get('Components', None):
	overwrite_comp_params = {int(k): v for k, v in params['Components'].items()}
	for comp_idx, comp_spec in enumerate(arch['Components']):
		comp_params = {}
		if comp_spec['Params']:
			comp_params = json.loads(comp_spec['Params'])
		if overwrite_comp_params.get(comp_idx, None):
			comp_params.update(json.loads(overwrite_comp_params[comp_idx]))
		comp_spec['Params'] = json.dumps(comp_params)

device = torch.device('cuda:0')
#device = torch.device('cpu')

# get train and val Datasets
print('loading datasets')
dataset_provider = skyhook_dataset.providers[params['Dataset']['Op']]
dataset_params = json.loads(params['Dataset']['Params'])
train_set, val_set = dataset_provider(url, datasets, dataset_params, train_split, valid_split)
datatypes = train_set.get_datatypes()

# get data augmentation steps
# this is a list of objects that provide forward() function
# we will apply the forward function on batches from DataLoader
print('loading data augmentations')
ds_augments = []
torch_augments = []
for spec in params['Augment']:
	cls_func = skyhook_augment.augmentations[spec['Op']]
	obj = cls_func(json.loads(spec['Params']), datatypes)
	if obj.pre_torch:
		ds_augments.append(obj)
	else:
		torch_augments.append(obj)

train_set.set_augments(ds_augments)
val_set.set_augments(ds_augments)

# apply data augmentation on validation set
# this is because some augmentations are random but we want a consistent validation set
# here we assume the validation set fits in system memory, but not necessarily GPU memory
# so we apply augmentation on CPU, whereas during training we will apply on GPU

train_params = json.loads(params['Train']['Params'])

print('preparing validation set')
val_loader = torch.utils.data.DataLoader(
	val_set,
	batch_size=batch_size,
	num_workers=4,
	collate_fn=val_set.collate_fn,
	# drop last unless we'd end up with 0 batches
	drop_last=len(val_set) > batch_size
)
val_batches = []
for batch in val_loader:
	for obj in torch_augments:
		batch = obj.forward(batch)
	val_batches.append(batch)

'''
batch = val_batches[0]
for i in range(32):
	im = batch[0][i, :, :, :].cpu().numpy().transpose(1, 2, 0)
	prefix = sum(batch[1]['counts'][0:i])
	detections = batch[1]['detections'][prefix:prefix+batch[1]['counts'][i]]
	for d in detections:
		cls, sx, sy, ex, ey = d
		sx = int(sx*im.shape[1])
		sy = int(sy*im.shape[0])
		ex = int(ex*im.shape[1])
		ey = int(ey*im.shape[0])
		im[sy:sy+2, sx:ex, :] = [255, 0, 0]
		im[ey-2:ey, sx:ex, :] = [255, 0, 0]
		im[sy:ey, sx:sx+2, :] = [255, 0, 0]
		im[sy:ey, ex-2:ex, :] = [255, 0, 0]
	skimage.io.imsave('/home/ubuntu/vis/{}.jpg'.format(i), im)
'''

'''
batch = val_batches[0]
for i in range(32):
	im1 = batch[0][i, :, :, :].cpu().numpy().transpose(1, 2, 0)
	im2 = (batch[1][i, 0, :, :].cpu().numpy() > 0).astype('uint8')*255
	skimage.io.imsave('/home/ubuntu/vis/{}_im.jpg'.format(i), im1)
	skimage.io.imsave('/home/ubuntu/vis/{}_mask.png'.format(i), im2)
'''

print('initialize model')
train_loader = torch.utils.data.DataLoader(
	train_set,
	batch_size=batch_size,
	shuffle=True,
	num_workers=4,
	collate_fn=train_set.collate_fn,
	# drop last unless we'd end up with 0 batches
	drop_last=len(train_set) > batch_size
)

for example_inputs in train_loader:
	break
util.inputs_to_device(example_inputs, device)
example_metadatas = train_set.get_metadatas(0)
net = model.Net(arch, comps, example_inputs, example_metadatas, device=device)
net.to(device)
learning_rate = train_params.get('LearningRate', 1e-3)
optimizer_name = train_params.get('Optimizer', 'adam')
if optimizer_name == 'adam':
	optimizer = torch.optim.Adam(net.parameters(), lr=learning_rate)
updated_lr = False

class StopCondition(object):
	def __init__(self, params):
		self.max_epochs = params.get('MaxEpochs', 0)

		# if score improves by less than score_epsilon for score_max_epochs epochs,
		# then we stop
		self.score_epsilon = params.get('ScoreEpsilon', 0)
		self.score_max_epochs = params.get('ScoreMaxEpochs', 25)

		# last score seen where we reset the score_epochs
		# this is less than the best_score only when score_epsilon > 0
		# (if a higher score is within epsilon of the last reset score)
		self.last_score = None
		# best score seen ever
		self.best_score = None

		self.epochs = 0
		self.score_epochs = 0

	def update(self, score):
		print(
			'epochs: {}/{} ... score: {}/{} (epochs since reset: {}/{}; best score: {})'.format(
			self.epochs, self.max_epochs, score, self.last_score, self.score_epochs, self.score_max_epochs, self.best_score
		))

		self.epochs += 1
		if self.max_epochs and self.epochs >= self.max_epochs:
			return True

		if self.best_score is None or score > self.best_score:
			self.best_score = score

		score_threshold = None
		if self.last_score is not None:
			score_threshold = self.last_score
			if self.score_epsilon is not None:
				score_threshold += self.score_epsilon
		if score_threshold is None or score > score_threshold:
			self.score_epochs = 0
			self.last_score = self.best_score
		else:
			self.score_epochs += 1
		if self.score_max_epochs and self.score_epochs >= self.score_max_epochs:
			return True

		return False

def save_model():
	# Save to a different filename first to reduce the chance of model being corrupted
	# if job is terminated.
	out_dir = os.path.join('data/items', str(out_dataset_id))
	torch.save(net.get_save_dict(), os.path.join(out_dir, 'model_.pt'))
	os.rename(os.path.join(out_dir, 'model_.pt'), os.path.join(out_dir, 'model.pt'))

class ModelSaver(object):
	def __init__(self, params):
		# either "latest" or "best"
		self.mode = params.get('Mode', 'best')

		self.best_score = None

	def update(self, net, score):
		should_save = False
		if self.mode == 'latest':
			should_save = True
		elif self.mode == 'best':
			if self.best_score is None or score > self.best_score:
				self.best_score = score
				should_save = True

		if should_save:
			save_model()

stop_condition = StopCondition(train_params['StopCondition'])
model_saver = ModelSaver(train_params['ModelSaver'])

rate_decay_params = train_params['RateDecay']
scheduler = None
if rate_decay_params['Op'] == 'step':
	scheduler = torch.optim.lr_scheduler.StepLR(optimizer, rate_decay_params['StepSize'], gamma=rate_decay_params['StepGamma'])
elif rate_decay_params['Op'] == 'plateau':
	scheduler = torch.optim.lr_scheduler.ReduceLROnPlateau(
		optimizer,
		mode='max',
		factor=rate_decay_params['PlateauFactor'],
		patience=rate_decay_params['PlateauPatience'],
		threshold=rate_decay_params['PlateauThreshold'],
		min_lr=rate_decay_params['PlateauMin']
	)

if params.get('Restore', None) and parent_models:
	for i, restore in enumerate(params['Restore']):
		if i >= len(parent_models):
			# could happen if user configured restore but then removed parent
			continue

		parent_model = parent_models[i]
		src_prefix = restore['SrcPrefix']
		dst_prefix = restore['DstPrefix']
		skip_prefixes = [prefix.strip() for prefix in restore['SkipPrefixes'].split(',') if prefix.strip()]
		print('restore model to', dst_prefix)
		# load save dict based on dataset ID
		fname = 'data/items/{}/model.pt'.format(parent_model['ID'])
		save_dict = torch.load(fname)
		# update the parameter names based on src/dst/skip prefixes
		state_dict = save_dict['model']
		new_dict = {}
		for k, v in state_dict.items():
			if not k.startswith(src_prefix):
				continue
			# check skip prefixes
			skip = False
			for prefix in skip_prefixes:
				if k.startswith(prefix):
					skip = True
					break
			if skip:
				continue
			# remove src_prefix and add dst_prefix
			k = k[len(src_prefix):]
			k = dst_prefix+k
			new_dict[k] = v

		missing_keys, unexpected_keys = net.load_state_dict(new_dict, strict=False)
		if missing_keys:
			print('... warning: got missing keys:', missing_keys)
		if unexpected_keys:
			print('... warning: got unexpected keys:', unexpected_keys)

epoch = 0

def get_loss_avgs(losses):
	loss_avgs = {}
	for k in losses[0].keys():
		loss_avgs[k] = numpy.mean([d[k] for d in losses])
	return loss_avgs

print('begin training')
save_model()
while True:
	train_losses = []
	net.train()
	for inputs in train_loader:
		util.inputs_to_device(inputs, device)
		for obj in torch_augments:
			inputs = obj.forward(inputs)
		optimizer.zero_grad()
		loss_dict, _ = net(*inputs[0:arch['NumInputs']], targets=inputs[arch['NumInputs']:])
		loss_dict['loss'].backward()
		optimizer.step()
		train_losses.append({k: v.item() for k, v in loss_dict.items()})

	val_losses = []
	net.eval()
	for inputs in val_batches:
		util.inputs_to_device(inputs, device)
		loss_dict, _ = net(*inputs[0:arch['NumInputs']], targets=inputs[arch['NumInputs']:])
		val_losses.append({k: v.item() for k, v in loss_dict.items()})

	train_loss_avgs = get_loss_avgs(train_losses)
	val_loss_avgs = get_loss_avgs(val_losses)

	json_loss = json.dumps({
		'train': train_loss_avgs,
		'val': val_loss_avgs,
	})
	print('jsonloss' + json_loss)

	val_loss = val_loss_avgs['loss']
	score = val_loss_avgs['score']

	if stop_condition.update(score):
		break
	model_saver.update(net, score)
	if scheduler is not None:
		scheduler.step(score)
		print('lr={}'.format(optimizer.param_groups[0]['lr']))

	epoch += 1
