import tensorflow
tf = tensorflow.compat.v1
tf.disable_eager_execution()
tf.experimental.output_all_intermediates(True)

import sys
sys.path.append('../')
# from RL.primal_dual_ddpg import *
# from RL.env import *

SIM_REAL_RATIO = 1

class input_config():
    batch_size = 32
    init_dual_lambda = 1
    state_dimension = 58
    action_dimension = 51
    clip_norm = 5.
    train_display_iter = 200
    model_save_path = './models/'
    # model_name = "sim_ddpg"
    # logdir = './logs/{}-{}-{}-{:.2f}/'.format(
    #     model_name, MAX_EP_STEPS, SIM_REAL_RATIO, init_dual_lambda)
    # log_path = logdir + 'saved_models/'
    log_path = "logs/nonpre_nonexp_" + str(SIM_REAL_RATIO) + "_pdddpg_summary"
    save_iter = 500
    log_iter = 100

def main():
    config = tf.ConfigProto(allow_soft_placement=True, log_device_placement=False)
    config.gpu_options.allow_growth = True

    # Set up summary writer
    summary_writer = tf.summary.FileWriter(input_config.log_path)

    summary_writer.close()

if __name__ == '__main__':
    main()