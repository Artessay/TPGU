import tensorflow.compat.v1 as tf
import numpy as np
from pprint import PrettyPrinter

import sys
sys.path.append('../')  # 将系统路径提高一层
from Simulator.data_model import BoilerDataSet

# tf.disable_v2_behavior()

# 定义参数，第一个是参数名称，第二个参数是默认值，第三个是参数描述
# 参数有四种取值：整数integer，浮点数float，字符串string，逻辑值boolean

# Data and model checkpoints directories
tf.app.flags.DEFINE_integer("display_iter", 200, "display_iter")
tf.app.flags.DEFINE_integer("save_log_iter", 100, "save_log_iter")

# Model params
tf.app.flags.DEFINE_integer("input_size", 109, "Input size")  # external_input + state + action
tf.app.flags.DEFINE_integer("output_size", 47, "Output size")  # state size

# Optimization
tf.app.flags.DEFINE_integer("num_steps", 10, "Number of steps")
tf.app.flags.DEFINE_float("val_ratio", 0.2, "valid ratio")      # @todo 在大规模数据集上可以改为0.1
tf.app.flags.DEFINE_integer("batch_size", 1, "The size of batch")
tf.app.flags.DEFINE_integer("max_epoch", 50, "Total training epoches")
tf.app.flags.DEFINE_float("grad_clip", 5., "Clip gradients at this value")
tf.app.flags.DEFINE_float("learning_rate", 0.001, "Initial learning rate at early stage. [0.001]")
tf.app.flags.DEFINE_float("learning_rate_decay", 0.95, "Decay rate of learning rate. [0.99]")
tf.app.flags.DEFINE_float("keep_prob", 1, "Keep probability of input data and dropout layer. [0.8]")
tf.app.flags.DEFINE_float("l2_weight", 0.0, "weight of l2 loss")

FLAGS = tf.app.flags.FLAGS

def main(_):
    np.random.seed(2022)    # 设置随机种子

    # tf.app.flags.FLAGS.__flags为包含了所有输入的列表
    # 当然，也可以单个查询，格式为：FLAGS.参数名
    # pprint模块负责以合适的格式打印便于阅读的行块。它使用换行和缩进以明确的方式打印数据。
    # PrettyPrinter().pprint(tf.app.flags.FLAGS.__flags)  # 打印参数列表 @debug

    # 设置GPU
    run_config = tf.ConfigProto()
    run_config.gpu_options.allow_growth = True

    # read data 读入数据
    boiler_dataset = BoilerDataSet(num_steps=FLAGS.num_steps, val_ratio=FLAGS.val_ratio)
    train_X, train_y = boiler_dataset.train_X, boiler_dataset.train_y
    valid_X, valid_y = boiler_dataset.valid_X, boiler_dataset.valid_y

    # 打印数据信息 @debug
    print('train samples: {0}'.format(len(train_X)))
    print('valid samples: {0}'.format(len(valid_X)))

if __name__ == '__main__':
    tf.app.run()    #tf.app.run()的作用：先处理flag解析，然后执行main函数