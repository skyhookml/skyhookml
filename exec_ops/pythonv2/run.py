import sys
sys.path.append('./python')
import skyhook.common as lib
import skyhook.io
from skyhook.op import Operator, per_frame

import importlib
import json

stdin = sys.stdin.detach()
meta = skyhook.io.read_json(stdin)
print('got meta', meta)

module_name = 'custom'
module_spec = importlib.util.spec_from_loader(module_name, loader=None)
module = importlib.util.module_from_spec(module_spec)
exec(meta['Code'], module.__dict__)
sys.modules[module_name] = module

operator = module.f(meta)

while True:
    try:
        request = skyhook.io.read_json(stdin)
    except EOFError:
        break

    id = request['RequestID']
    name = request['Name']
    if request['JSON']:
        params = json.loads(request['JSON'])
    else:
        params = None

    response = None
    if name == 'parallelism':
        response = operator.parallelism()
    elif name == 'get_tasks':
        response = operator.get_tasks(params)
    elif name == 'apply':
        operator.apply(params)

    packet = {
        'RequestID': id,
    }
    if response is not None:
        packet['JSON'] = json.dumps(response)
    print('skjson'+json.dumps(packet), flush=True)
