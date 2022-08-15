import tensorflow.compat.v1 as tf


# 定义参数，第一个是参数名称，第二个参数是默认值，第三个是参数描述
flags = tf.app.flags    # 简写

# Data and model checkpoints directories
flags.DEFINE_integer("display_iter", 200, "display_iter")
flags.DEFINE_integer("save_log_iter", 100, "save_log_iter")

# Model params
flags.DEFINE_integer("input_size", 109, "Input size")  # external_input + state + action
flags.DEFINE_integer("output_size", 47, "Output size")  # state size

# Optimization
flags.DEFINE_integer("num_steps", 10, "Num of steps")
flags.DEFINE_integer("batch_size", 1, "The size of batch")
flags.DEFINE_integer("max_epoch", 50, "Total training epoches")
flags.DEFINE_float("grad_clip", 5., "Clip gradients at this value")
flags.DEFINE_float("learning_rate", 0.001, "Initial learning rate at early stage. [0.001]")
flags.DEFINE_float("learning_rate_decay", 0.95, "Decay rate of learning rate. [0.99]")
flags.DEFINE_float("keep_prob", 1, "Keep probability of input data and dropout layer. [0.8]")
flags.DEFINE_float("l2_weight", 0.0, "weight of l2 loss")

FLAGS = flags.FLAGS

def main(_):
    print(FLAGS.display_iter)

if __name__ == '__main__':
    tf.app.run()    #tf.app.run()的作用：先处理flag解析，然后执行main函数