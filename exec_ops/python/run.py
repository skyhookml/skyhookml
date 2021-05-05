import sys
sys.path.append('./python')
import skyhook.common as lib

import importlib

def f(meta):
	module_name = 'custom'
	module_spec = importlib.util.spec_from_loader(module_name, loader=None)
	module = importlib.util.module_from_spec(module_spec)
	exec(meta['Code'], module.__dict__)
	sys.modules[module_name] = module

	return module.f(meta)

lib.run(f)
