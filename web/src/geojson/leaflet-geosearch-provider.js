let LatLonProvider = (provider) => {
	let wrapper = {};
	wrapper.getParamString = (params) => {
		return provider.getParamString(params);
	};
	wrapper.getUrl = (url, params) => {
		return provider.getUrl(url, params);
	}
	wrapper.search = async (options) => {
		// Is this in "LAT, LON" format?
		const re = /^ *(-?[0-9\.]+) *, *(-?[0-9\.]+) *$/;
		let result = re.exec(options.query);
		if(!result) {
			return provider.search(options);
		}
		let lat = parseFloat(result[1]);
		let lon = parseFloat(result[2]);

		return [{
			x: lon,
			y: lat,
			label: options.query,
			bounds: [
				[lat-0.01, lon-0.01],
				[lat+0.01, lon+0.01],
			],
			raw: {},
		}];
	};
	return wrapper;
};
export default LatLonProvider;
