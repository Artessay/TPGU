import tensorflow as tf
import numpy as np

import sys
sys.path.append('../')  # 将系统路径提高一层

from Simulator.data_model import BoilerDataSet
from Simulator.sim_rnn_model import SimulatorRNNModel

class sim_config(object):
    num_steps = 10
    valid_ratio = 0.2

    input_size = 202
    num_neurons = 160
    num_layers = 3
    output_size = 158

    learning_rate = 0.001
    learning_rate_decay = 0.95

    max_epoch = 50
    batch_size = 1

    save_log_iter = 10
    display_iter = 20

class cell_config(object):
    """ Simulator Cell config """
    # list, [coaler_num_units, burner_num_units, steamer_num_units]
    num_units = [128, 128, 64]   # 各层神经元数量

    # data is [external_input, state(coaler, burner, steamer), action(coaler, burner, steamer)]
    # data size is [11, 147(68, 62, 17), 44(21, 19, 4)]. Total size is 202
    external_state_pos = 0
    external_state_size = 11
    
    coaler_state_pos = external_state_pos + external_state_size
    coaler_state_size = 68
    burner_state_pos = coaler_state_pos + coaler_state_size
    burner_state_size = 62
    steamer_state_pos = burner_state_pos + burner_state_size
    steamer_state_size = 17

    coaler_action_pos = steamer_state_pos + steamer_state_size
    coaler_action_size = 21
    burner_action_pos = coaler_action_pos + coaler_action_size
    burner_action_size = 19
    steamer_action_pos = burner_action_pos + burner_action_size
    steamer_action_size = 4

def reset_random_seed(seed=2022):
    tf.random.set_seed(seed)
    np.random.seed(seed)

def fit_and_evaluate(model, train_X, train_y, valid_X, valid_y, epochs=500):
    callback_list = [
        tf.keras.callbacks.EarlyStopping(
            monitor="val_loss", 
            patience=50, 
            restore_best_weights=True
        ),
        tf.keras.callbacks.ModelCheckpoint(
            filepath='./logs/LSTM/saved_models/model-{epoch:02d}-{val_loss:.4f}.h5',
            monitor="val_loss",
            verbose=1,
            save_weights_only=True,
            save_best_only=True,
        ),
        tf.keras.callbacks.TensorBoard(log_dir='./logs/')
    ]   
    
    history = model.fit(
        x=train_X, y=train_y,
        epochs=epochs, 
        validation_data=(valid_X, valid_y),
        callbacks=callback_list)
    valid_loss, valid_mae = model.evaluate(x=valid_X, y=valid_y) # Returns the loss value & metrics values for the model in test mode
    return valid_mae * 1e6  # valid mean absolute error

def main(sim_config):
    reset_random_seed()    # set random seed

    num_steps = sim_config.num_steps
    valid_ratio = sim_config.valid_ratio
    max_epoch = sim_config.max_epoch

    # read data
    boiler_dataset = BoilerDataSet(num_steps=num_steps, val_ratio=valid_ratio)
    train_X, train_y = boiler_dataset.train_X, boiler_dataset.train_y
    valid_X, valid_y = boiler_dataset.valid_X, boiler_dataset.valid_y

    # prepare model
    model = SimulatorRNNModel(sim_config=sim_config)
    model.summary()
    
    fit_and_evaluate(model, train_X, train_y, valid_X, valid_y, max_epoch)

if __name__ == '__main__':
    main(sim_config=sim_config())