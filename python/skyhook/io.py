import json
import numpy
import struct

def read_json(f):
	buf = f.read(4)
	if not buf:
		raise EOFError
	(hlen,) = struct.unpack('>I', buf[0:4])
	json_data = f.read(hlen)
	return json.loads(json_data.decode('utf-8'))

def read_array(f, dims=None, dt=None):
	header = read_json(f)

	if dims is None:
		dims = header

	size = header['Length']*header['BytesPerElement']
	buf = f.read(size)
	return numpy.copy(numpy.frombuffer(buf, dtype=dt).reshape((header['Length'], dims['Height'], dims['Width'], dims['Channels'])))

def read_datas(f, dtypes, metadatas):
	datas = []
	for i, t in enumerate(dtypes):
		if t == 'image' or t == 'video' or t == 'geoimage':
			datas.append(read_array(f, dt=numpy.dtype('uint8')))
		elif t == 'array':
			dt = numpy.dtype(metadatas[i]['Type'])
			dt = dt.newbyteorder('>')
			dims = metadatas[i]
			datas.append(read_array(f, dims=dims, dt=dt))
		else:
			datas.append(read_json(f))
	return datas


def write_json(f, x):
	s = json.dumps(x).encode()
	f.write(struct.pack('>I', len(s)))
	f.write(s)

def write_array(f, x):
	if x is None:
		write_json(f, {
			'Length': 0,
			'BytesPerElement': 0,
		})
		return

	write_json(f, {
		'Width': x.shape[2],
		'Height': x.shape[1],
		'Channels': x.shape[3],
		'Length': x.shape[0],
		'BytesPerElement': x.shape[1]*x.shape[2]*x.shape[3]*x.dtype.itemsize,
	})
	dt = numpy.dtype(x.dtype.name)
	dt = dt.newbyteorder('>')
	f.write(x.astype(dt, copy=False).tobytes())

def write_datas(f, dtypes, datas):
	for i, t in enumerate(dtypes):
		if t == 'image' or t == 'video' or t == 'array' or t == 'geoimage':
			write_array(f, datas[i])
		else:
			write_json(f, datas[i])
