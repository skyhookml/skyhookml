augmentations = {}

import skyhook.pytorch.augment.random_resize as random_resize
augmentations['random_resize'] = random_resize.RandomResize
