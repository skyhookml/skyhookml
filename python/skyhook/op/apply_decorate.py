from skyhook.op.op import Operator

class ApplyDecorateOperator(Operator):
	def __init__(self, meta_packet):
		super(ApplyDecorateOperator, self).__init__(meta_packet)
		# Function must be set after initialization.
		self.f = None

	def apply(self, task):
		self.f(self, task)

def apply_decorate(f):
	def wrap(meta_packet):
		op = ApplyDecorateOperator(meta_packet)
		op.f = f
		return op
	return wrap
