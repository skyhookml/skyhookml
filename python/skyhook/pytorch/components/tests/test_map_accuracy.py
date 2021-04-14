import numpy
from skyhook.pytorch.components import map_accuracy
import torch

import unittest

# get_iou
class TestGetIou(unittest.TestCase):
	# Test two predicted boxes and two gt boxes.
	def test_two(self):
		pred = numpy.array([
			[0.1, 0.1, 0.2, 0.2],
			[0.5, 0.5, 0.6, 0.6],
		], dtype='float32')
		gt = numpy.array([
			[0.15, 0.15, 0.2, 0.2],
			[0.5, 0.5, 0.6, 0.6],
		], dtype='float32')
		iou_matrix = map_accuracy.get_iou(pred, gt)
		expect_matrix = numpy.array([
			[0.25, 0],
			[0, 1],
		], dtype='float32')
		self.assertTrue(numpy.max(numpy.abs(iou_matrix - expect_matrix)) < 1e-6)

if __name__ == '__main__':
	unittest.main()
