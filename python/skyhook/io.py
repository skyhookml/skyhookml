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

def read_array(f, channels=None, dt=None):
	header = read_json(f)

	if channels is None:
		channels = header['Channels']
	if dt is None:
		dt = numpy.dtype(header['Type'])
		dt = dt.newbyteorder('>')

	size = header['Length']*header['Width']*header['Height']*channels*dt.itemsize
	buf = f.read(size)
	return numpy.frombuffer(buf, dtype=dt).reshape((header['Length'], header['Height'], header['Width'], channels))

def read_datas(f, dtypes):
	datas = []
	for t in dtypes:
		if t == 'image' or t == 'video':
			datas.append(read_array(f, channels=3, dt=numpy.dtype('uint8')))
		elif t == 'array':
			datas.append(read_array(f))
		elif t == 'geoimage':
			header = read_json(f)
			metadata = header['Metadata']
			if header['Width'] > 0:
				buf = f.read(header['Width']*header['Height']*3)
				im = numpy.frombuffer(buf, dtype='uint8').reshape((header['Height'], header['Width'], 3))
			else:
				im = None
			datas.append({
				'Metadata': metadata,
				'Image': im,
			})
		else:
			datas.append(read_json(f))
	return datas


def write_json(f, x):
	s = json.dumps(x).encode()
	f.write(struct.pack('>I', len(s)))
	f.write(s)

def write_array(f, x):
	write_json(f, {
		'Length': x.shape[0],
		'Width': x.shape[2],
		'Height': x.shape[1],
		'Channels': x.shape[3],
		'Type': x.dtype.name,
	})
	dt = numpy.dtype(x.dtype.name)
	dt = dt.newbyteorder('>')
	f.write(x.astype(dt, copy=False).tobytes())

def write_datas(f, dtypes, datas):
	for i, t in enumerate(dtypes):
		if t == 'image' or t == 'video' or t == 'array':
			write_array(f, datas[i])
		elif t == 'geoimage':
			metadata = datas[i]['Metadata']
			im = datas[i]['Image']
			if im is None:
				width, height = 0, 0
			else:
				width, height = im.shape[1], im.shape[0]
			write_json(f, {
				'Metadata': metadata,
				'Width': width,
				'Height': height,
			})
			if im:
				f.write(im.tobytes())
		else:
			write_json(f, datas[i])
