augmentations = {}

import skyhook.pytorch.augment.random_resize as random_resize
augmentations['random_resize'] = random_resize.RandomResize

import skyhook.pytorch.augment.crop as crop
augmentations['crop'] = crop.Crop

import skyhook.pytorch.augment.flip as flip
augmentations['flip'] = flip.Flip
