import sys
sys.path.append('./python')

import skyhook.common as lib
from skyhook.op import per_frame

import cv2
import geojson
import json
import numpy
import skimage.io
import sys

params = {
	'DenoiseSize': 5,
	'GrowSize': 5,
	'SimplifyThreshold': 0.01,
}

@per_frame
def f(arr_data):
	def get_polygons(arr, kernel_size_denoise, kernel_size_grow, simplify_threshold):
		mask = (arr[:, :, 1] > 0.5).astype('uint8')

		# denoise
		if kernel_size_denoise is not None:
			struct = cv2.getStructuringElement(cv2.MORPH_ELLIPSE, (kernel_size_denoise, kernel_size_denoise))
			mask = cv2.morphologyEx(mask, cv2.MORPH_OPEN, struct)

		# grow
		if kernel_size_grow is not None:
			struct = cv2.getStructuringElement(cv2.MORPH_ELLIPSE, (kernel_size_grow, kernel_size_grow))
			mask = cv2.morphologyEx(mask, cv2.MORPH_CLOSE, struct)

		# countors
		multipolygons, hierarchy = cv2.findContours(mask, cv2.RETR_TREE, cv2.CHAIN_APPROX_SIMPLE)

		if hierarchy is None:
			return []

		assert len(hierarchy) == 1, "always single hierarchy for all polygons in multipolygon"
		hierarchy = hierarchy[0]
		assert len(multipolygons) == len(hierarchy), "polygons and hierarchy in sync"

		# simplify
		def simplify(polygon, eps):
			epsilon = eps * cv2.arcLength(polygon, closed=True)
			return cv2.approxPolyDP(polygon, epsilon=epsilon, closed=True)
		polygons = [simplify(polygon, simplify_threshold) for polygon in multipolygons]

		# only keep the outermost polygons?

		def parents_in_hierarchy(node, tree):
			def parent(n):
				return n[3]
			at = tree[node]
			up = parent(at)
			while up != -1:
				index = up
				at = tree[index]
				up = parent(at)
				assert index != node, "upward path does not include starting node"
				yield

		shapes = []
		for i, (polygon, node) in enumerate(zip(polygons, hierarchy)):
			if len(polygon) < 3:
				continue

			_, _, _, parent_idx = node
			ancestors = list(parents_in_hierarchy(i, hierarchy))
			if len(ancestors) > 0:
				continue

			shape = {
				'Type': 'polygon',
				'Points': [[int(p[0][0]), int(p[0][1])] for p in polygon],
			}
			shapes.append(shape)


		return shapes

	probs = arr_data['Data']
	shapes = get_polygons(
		probs,
		kernel_size_denoise=params['DenoiseSize'],
		kernel_size_grow=params['GrowSize'],
		simplify_threshold=params['SimplifyThreshold']
	)
	return {
		'Metadata': {
			'CanvasDims': [probs.shape[1], probs.shape[0]],
		},
		'Data': shapes,
	}

def handle_func(meta):
	decoded_params = json.loads(meta['Code'])
	params.update(decoded_params)
	return f(meta)

lib.run(handle_func)
