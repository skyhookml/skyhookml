import sys
sys.path.append('./')
import skyhook_pylib as lib

import io
import json
import math
import numpy
import os
import os.path
import skimage.io
import struct

user_func = None

# user_func will be defined by the exec call in meta_func
def callback(*args):
    return user_func(*args)

def meta_func(meta):
    global user_func
    # code should define a function "f"
    locals = {}
    exec(meta['Code'], None, locals)
    user_func = locals['f']

lib.run(callback, meta_func)
