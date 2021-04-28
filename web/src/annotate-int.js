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
	destroyed: function() {
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
	on_image_loaded: function() {
		// if range is 0, focus on the text input
		Vue.nextTick(() => {
			if(this.params.Range == 0 && this.$refs.input) {
				this.$refs.input.focus();
			}
		});
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
<form class="row g-1 align-items-center" v-on:submit.prevent="saveParams">
	<div class="col-auto">
		<label class="my-1 mx-1">Range</label>
	</div>
	<div class="col-auto">
		<input type="text" class="form-control my-1 mx-1" v-model="params.Range" />
	</div>
	<div class="col-auto">
		<button type="submit" class="btn btn-primary my-1 mx-1">Save Settings</button>
	</div>
</form>
		`,
		im_below: `
<div v-if="imageDims != null" class="row g-1 align-items-center mb-2">
	<div class="col-auto">
		<span v-if="ints[frameIdx] != ''">Current Label: {{ ints[frameIdx] }}</span>
	</div>
	<template v-if="parseInt(params.Range) > 0">
		<div v-for="i in parseInt(params.Range)" class="col-auto">
			<button v-on:click="submitButton(i-1)" type="button" class="btn btn-primary">{{ i-1 }}</button>
		</div>
	</template>
	<template v-else>
		<div class="col-auto">
			<form class="row g-1 align-items-center" v-on:submit.prevent="submitInput">
				<div class="col-auto">
					<input type="text" class="form-control" v-model="ints[frameIdx]" ref="input" />
				</div>
				<div class="col-auto">
					<button type="submit" class="btn btn-primary">Label</button>
				</div>
			</form>
		</div>
	</template>
</div>
		`,
	},
});
