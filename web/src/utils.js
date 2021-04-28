function request(comp, method, endpoint, params, successFunc, completeFunc, opts) {
	var args = {
		type: method,
		url: endpoint,
		error: (req, status, errorMsg) => {
			if(!comp) {
				return;
			}
			if(req && req.responseText) {
				errorMsg = req.responseText;
			}
			if(comp.setError) {
				comp.setError(errorMsg);
			} else if(comp.$globals.app) {
				comp.$globals.app.setError(errorMsg);
			}
		},
	};
	if(params) {
		args.data = params;
		if(typeof(args.data) === 'string') {
			args.processData = false;
		}
	}
	if(successFunc) {
		args.success = successFunc;
	}
	if(completeFunc) {
		args.complete = completeFunc;
	}
	if(opts) {
		if(opts.dataType) {
			args.dataType = opts.dataType;
		}
		if(opts.error) {
			// override the error handler we set above
			args.error = opts.error;
		}
	}
	return $.ajax(args);
}

// Returns Promise that waits for a job to complete before proceeding.
function waitForJob(jobID) {
	return new Promise((resolve, reject) => {
		let interval;
		let checkFunc = () => {
			request(this, 'GET', '/jobs/'+jobID, null, (job) => {
				if(!job.Done) {
					return;
				}
				clearInterval(interval);
				if(job.Error) {
					reject(job);
					return;
				}
				resolve(job);
			});
		};
		interval = setInterval(checkFunc, 1000);
	});
};

export default {
	request: request,
	waitForJob: waitForJob,
};
