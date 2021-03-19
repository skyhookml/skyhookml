import utils from './utils.js';
import AnnotateGenericUI from './annotate-generic-ui.js';

export default AnnotateGenericUI({
	data: function() {
		return {
			params: null,

			// we store the ints as strings
			ints: [],
		};
	},
	on_created_ready: function() {
		let params;
		try {
			params = JSON.parse(this.annoset.Params);
		} catch(e) {}
		if(!params) {
			params = {};
		}
		if(!params.Range) {
			params.Range = 2;
		}
		this.params = params;
	},
	mounted: function() {
		this.keypressHandler = (e) => {
			if(document.activeElement.tagName == 'INPUT') {
				return;
			}

			// keycode 48 through 57 are 0 through 9
			if(e.keyCode < 48 || e.keyCode > 57) {
				return;
			}
			let label = parseInt(e.keyCode) - 48;
			this.ints[this.frameIdx] = label.toString();
			this.finishFrame();
		};
		this.$parent.$on('keypress', this.keypressHandler);
	},
	unmounted: function() {
		this.$parent.$off('keypress', this.keypressHandler);
		this.keypressHandler = null;
	},
	on_update: function() {
		this.ints = [];
		for(let i = 0; i < this.numFrames; i++) {
			this.ints.push('');
		}
	},
	on_item_data: function(data) {
		if(data.length == 0) {
			return;
		}
		this.ints = data.map((x) => x.toString());
	},
	getAnnotateData: function() {
		let data = this.ints.map((str) => {
			if(isNaN(parseInt(str))) {
				return -1;
			}
			return parseInt(str);
		});
		let metadata = null;
		return [data, metadata];
	},
	methods: {
		submitInput: function() {
			this.finishFrame();
		},
		submitButton: function(x) {
			this.ints[this.frameIdx] = x.toString();
			this.finishFrame();
		},
		saveParams: function() {
			let request = {
				Params: JSON.stringify(this.params),
			}
			utils.request(this, 'POST', '/annotate-datasets/'+this.annoset.ID, JSON.stringify(request));
		},
	},
	template: {
		params: `
<form class="form-inline" v-on:submit.prevent="saveParams">
	<label class="my-1 mx-1">Range</label>
	<input type="text" class="form-control my-1 mx-1" v-model="params.Range" />

	<button type="submit" class="btn btn-primary my-1 mx-1">Save Settings</button>
</form>
		`,
		im_below: `
<div v-if="imageDims != null" class="form-row align-items-center">
	<span v-if="ints[frameIdx] != ''">Current Label: {{ ints[frameIdx] }}</span>
	<template v-if="parseInt(params.Range) > 0">
		<div v-for="i in parseInt(params.Range)">
			<button v-on:click="submitButton(i-1)" type="button" class="btn btn-primary">{{ i-1 }}</button>
		</div>
	</template>
	<template v-else>
		<div class="col-auto">
			<form class="form-inline" v-on:submit.prevent="submitInput">
				<input type="text" class="form-control" v-model="ints[frameIdx]" />
				<button type="submit" class="btn btn-primary">Label</button>
			</form>
		</div>
	</template>
</div>
		`,
	},
});
